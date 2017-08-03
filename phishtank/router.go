package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

// appContext holds the web context for the handlers
type appContext struct {
	loadedData *fileData
}

// query the given file and return the resulting JSON
func (ac *appContext) query(w http.ResponseWriter, r *http.Request) {
	p := getPathParams(r)
	f := p.ByName("file")
	q := r.FormValue("q")
	if f == "" || q == "" {
		log.Info("Request missing file or query")
		writeError(w, ErrBadRequest)
		return
	}
	data := ac.loadedData.Get(f)
	if data == nil {
		log.Info("File is not loaded")
		writeError(w, ErrNotLoaded)
		return
	}
	phish := ac.loadedData.GetURL(f, q)
	if phish == nil {
		log.Infof("URL %s not found", q)
		writeError(w, ErrNotFound)
		return
	}
	writeJSON(phish, w)
}

// Errors is a list of errors
type Errors struct {
	Errors []*Error `json:"errors"`
}

// Error holds the info about a web error
type Error struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title,omitempty"`
	Detail string `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	return e.Title + ":" + e.Detail
}

// writeJSON writes ok to reply
func writeJSON(data interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error to the reply
func writeError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

var (
	// ErrBadRequest is a generic bad request
	ErrBadRequest = &Error{"bad_request", 400, "Bad request", "Request is missing parts"}
	// ErrNotLoaded is a generic bad request
	ErrNotLoaded = &Error{"not_loaded", 400, "File not loaded", "Requested file is not loaded"}
	// ErrNotFound URL not found
	ErrNotFound = &Error{"not_found", 404, "Not found", "The requested URL was not found"}
	// ErrNotAcceptable wrong accept header
	ErrNotAcceptable = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/json'."}
	// ErrUnsupportedMediaType wrong media type
	ErrUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/json'."}
	// ErrInternalServer if things go wrong on our side
	ErrInternalServer = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)

type requestContextKey string

const (
	contextBody   = requestContextKey("body")
	contextParams = requestContextKey("params")
)

func setRequestContext(r *http.Request, key requestContextKey, val interface{}) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, val))
}

func getRequestBody(r *http.Request) interface{} {
	return r.Context().Value(contextBody)
}

func getPathParams(r *http.Request) httprouter.Params {
	return r.Context().Value(contextParams).(httprouter.Params)
}

// router handles the web requests routing
type router struct {
	*httprouter.Router
	staticHandlers alice.Chain
	commonHandlers alice.Chain
	appContext     *appContext
}

// Get handles GET requests
func (r *router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

// Post handles POST requests
func (r *router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

// Put handles PUT requests
func (r *router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

// Delete handles DELETE requests
func (r *router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

// New creates a new router
func newRouter(appC *appContext) *router {
	r := &router{Router: httprouter.New()}
	r.appContext = appC
	r.staticHandlers = alice.New(loggingHandler, recoverHandler, clickjackingHandler)
	r.commonHandlers = r.staticHandlers.Append(acceptHandler)
	r.registerHandlers()
	return r
}

// registerHandlers for the available actions
func (r *router) registerHandlers() {
	r.Get("/query/:file", r.commonHandlers.ThenFunc(r.appContext.query))
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		r = setRequestContext(r, contextParams, ps)
		h.ServeHTTP(w, r)
	}
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// serve - creates the relevant listeners
func (r *router) serve() {
	var err error
	if options.SSL.Cert != "" {
		addr := options.Address
		if addr == "" {
			addr = ":https"
		}
		server := &http.Server{Addr: options.Address, Handler: r}
		config, err := getTLSConfig()
		if err != nil {
			log.Fatal(err)
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
		err = server.Serve(tlsListener)
	} else {
		err = http.ListenAndServe(options.Address, r)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// getTLSConfig ...
func getTLSConfig() (config *tls.Config, err error) {
	certs := make([]tls.Certificate, 1)
	certs[0], err = tls.X509KeyPair([]byte(options.SSL.Cert), []byte(options.SSL.Key))
	if err != nil {
		return nil, err
	}
	config = &tls.Config{
		NextProtos:               []string{"http/1.1"},
		MinVersion:               tls.VersionTLS12,
		Certificates:             certs,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		},
	}
	return
}
