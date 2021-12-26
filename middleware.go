package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type LoggingHandler struct {
	writer  io.Writer
	handler http.Handler
}

type StatusRecorder struct {
	http.ResponseWriter
	http.Hijacker
	status int
}

func (w *StatusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}

func (r *StatusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (h LoggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := time.Now()
	url := *req.URL
	recorder := &StatusRecorder{w, w.(http.Hijacker), 200}
	h.handler.ServeHTTP(recorder, req)
	if req.MultipartForm != nil {
		req.MultipartForm.RemoveAll()
	}
	dur := time.Now().Sub(t)
	log.Printf("%s %s %d %s", dur.String(), req.Method, recorder.status, &url)
}

func AddLogging(out io.Writer, h http.Handler) http.Handler {
	return LoggingHandler{out, h}
}

func ItemsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		rc := getRequestContext(req)
		if ok := rc.isAuthorized(); !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := req.Context()
		ctx = context.WithValue(ctx, "groceries", rc)
		groceriesRequest := req.Clone(ctx)
		next.ServeHTTP(w, groceriesRequest)
	})
}
