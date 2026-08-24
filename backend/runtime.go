package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var requestSequence uint64

func serveAddress(address string, handler http.Handler) error {
	return serveHTTP(newEnterpriseServer(address, handler))
}

func serveHTTP(server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals:
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownContext)
	}
}

func newEnterpriseServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           opsEnterpriseMiddleware(requestIDMiddleware(recoveryMiddleware(handler))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req-%d", atomic.AddUint64(&requestSequence, 1))
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rr := &recoverResponseWriter{ResponseWriter: w}
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			requestID := w.Header().Get("X-Request-ID")
			log.Printf("panic recovered request_id=%s method=%s path=%s panic=%v", requestID, r.Method, r.URL.Path, recovered)
			// If the handler already flushed a response before panicking, leave
			// the partial response in place: writing a second WriteHeader would
			// be logged as a superfluous call and the body is already committed.
			if rr.wrote {
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":     "internal server error",
				"requestId": requestID,
			})
		}()
		next.ServeHTTP(rr, r)
	})
}

// recoverResponseWriter tracks whether the wrapped handler has started writing
// the response so the recovery middleware can decide whether it is still safe
// to synthesize a 500 after a panic.
type recoverResponseWriter struct {
	http.ResponseWriter
	wrote bool
}

func (r *recoverResponseWriter) WriteHeader(statusCode int) {
	r.wrote = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *recoverResponseWriter) Write(b []byte) (int, error) {
	r.wrote = true
	return r.ResponseWriter.Write(b)
}

// Unwrap exposes the underlying ResponseWriter so middleware that needs the
// original (e.g. http.Flusher / http.Hijacker checks) still works.
func (r *recoverResponseWriter) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
