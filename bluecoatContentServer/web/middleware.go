package web

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-errors/errors"
)

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.WithField("error", err).Warn("Recovered from error")
				log.Error(errors.Wrap(err, 2).ErrorStack())
				writeError(w, ErrInternalServer)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingResponseWriter{w, 200}
		t1 := time.Now()
		next.ServeHTTP(lw, r)
		t2 := time.Now()
		log.Infof("[%s] %q %v %v", r.Method, r.URL.String(), lw.status, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func acceptHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept"), "application/json") {
			log.Warn("Request without accept header received")
			writeError(w, ErrNotAcceptable)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeHandler(next http.Handler, contentType string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if !strings.Contains(ct, contentType) {
			log.Warnf("Request without proper content type received. Got: %s, Expected: %s", ct, contentType)
			writeError(w, ErrUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func jsonContentTypeHandler(next http.Handler) http.Handler {
	return contentTypeHandler(next, "application/json")
}

func bodyHandler(v interface{}) func(http.Handler) http.Handler {
	t := reflect.TypeOf(v)

	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			val := reflect.New(t).Interface()
			err := json.NewDecoder(r.Body).Decode(val)

			if err != nil {
				log.WithFields(log.Fields{"body": r.Body, "err": err}).Warn("Error handling body")
				writeError(w, ErrBadRequest)
				return
			}

			if next != nil {
				r = setRequestContext(r, contextBody, val)
				next.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}

const (
	// xFrameOptionsHeader is the name of the x frame header
	xFrameOptionsHeader = `X-Frame-Options`
)

// Handle Clickjacking protection
func clickjackingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(xFrameOptionsHeader, "DENY")
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func (ac *AppContext) authHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			log.Info("Unable to parse basic auth")
			writeError(w, ErrAuth)
			return
		}
		u, err := ac.r.User(username)
		if err != nil {
			log.WithError(err).Infof("Unable to load user %s", username)
			ac.handleLoginError(w, username)
			return
		}
		if !u.ValidPassword(password) {
			log.WithError(err).Infof("Invalid password for user %s", username)
			ac.handleLoginError(w, username)
			return
		}
		// successful login need to reset
		ac.resetBruteForce(username)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
