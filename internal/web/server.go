package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"box-link/internal/app"
)

//go:embed static/*
var embeddedStatic embed.FS

type Server struct {
	application *app.App
	addr        string
}

func New(application *app.App, addr string) *Server {
	return &Server{application: application, addr: addr}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	registerAPI(mux, s.application)

	staticFS, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return fmt.Errorf("prepare static files: %w", err)
	}

	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/app.css", fileServer)
	mux.Handle("/app.js", fileServer)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFileFS(w, r, staticFS, "index.html")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	}))

	httpServer := &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	s.application.Log.Infof("web UI listening on http://%s", s.addr)

	err = <-errCh
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return err
}
