package server

import (
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"text/template"
	"time"

	"github.com/robherley/etherlighter/internal/config"
	"github.com/robherley/etherlighter/internal/device"
)

func New(cfg *config.Config, files fs.FS, client *device.Client) (*http.Server, error) {
	indexTemplate := template.Must(template.ParseFS(files, "index.go.html"))

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		if cfg.DevMode {
			indexTemplate = template.Must(template.ParseFS(files, "index.go.html"))
		}

		info, err := client.Info()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := indexTemplate.Execute(w, info); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("POST /api/port-colors", func(w http.ResponseWriter, r *http.Request) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var colors []device.PortColor
		if err := json.Unmarshal(bytes, &colors); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := client.SetPortColors(colors); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /api/mode", func(w http.ResponseWriter, r *http.Request) {
		bytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var req struct {
			Mode device.Mode `json:"mode"`
		}

		if err := json.Unmarshal(bytes, &req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := client.SetMode(req.Mode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	return &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: withLogger(mux),
	}, nil
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == -1 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func withLogger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{w, -1}
		h.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"duration", time.Since(start),
			"status", rw.status,
		)
	})
}
