package web

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/tools/bluecoatContentServer/conf"
	"github.com/demisto/tools/bluecoatContentServer/domain"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
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

// Router

// Router handles the web requests routing
type Router struct {
	*httprouter.Router
	staticHandlers alice.Chain
	commonHandlers alice.Chain
	authHandlers   alice.Chain
	fileHandlers   alice.Chain
	appContext     *AppContext
}

// Get handles GET requests
func (r *Router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

// Post handles POST requests
func (r *Router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

// Put handles PUT requests
func (r *Router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

// Delete handles DELETE requests
func (r *Router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

// New creates a new router
func New(appC *AppContext) *Router {
	initBruteForceMap(false)
	r := &Router{Router: httprouter.New()}
	r.appContext = appC
	r.staticHandlers = alice.New(loggingHandler, recoverHandler, clickjackingHandler)
	r.commonHandlers = r.staticHandlers.Append(acceptHandler)
	r.authHandlers = r.commonHandlers.Append(appC.authHandler)
	r.fileHandlers = r.staticHandlers.Append(appC.authHandler)
	r.registerHandlers()
	return r
}

// registerHandlers for the available actions
func (r *Router) registerHandlers() {
	// Download local rules DB
	r.Get("/db", r.fileHandlers.ThenFunc(r.appContext.dbHandler))
	r.Post("/db/add", r.authHandlers.Append(jsonContentTypeHandler, bodyHandler(domain.Rule{})).ThenFunc(r.appContext.addRule))
	r.Post("/db/remove", r.authHandlers.Append(jsonContentTypeHandler, bodyHandler(domain.Rule{})).ThenFunc(r.appContext.removeRule))
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

// Serve - creates the relevant listeners
func (r *Router) Serve() {
	var err error
	if conf.Options.SSL.Cert != "" {
		addr := conf.Options.Address
		if addr == "" {
			addr = ":https"
		}
		server := &http.Server{Addr: conf.Options.Address, Handler: r}
		config, err := GetTLSConfig()
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
		err = http.ListenAndServe(conf.Options.Address, r)
	}
	if err != nil {
		log.Fatal(err)
	}
}

// GetTLSConfig ...
func GetTLSConfig() (config *tls.Config, err error) {
	certs := make([]tls.Certificate, 1)
	certs[0], err = tls.X509KeyPair([]byte(conf.Options.SSL.Cert), []byte(conf.Options.SSL.Key))
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
