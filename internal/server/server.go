package server

import (
	"log"
	"net/http"

	"ssg.nakurai/assets"
)

// Serve starts the dev HTTP server on addr (e.g. ":8088").
// It serves static files from dir, the SSE livereload endpoint, and the
// livereload.js script.
func Serve(dir, addr string, hub *Hub) error {
	mux := http.NewServeMux()

	// Livereload script
	lrJS, err := assets.FS.ReadFile("livereload/livereload.js")
	if err != nil {
		return err
	}
	mux.HandleFunc("/__livereload.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write(lrJS)
	})

	// SSE endpoint
	mux.Handle("/__livereload", hub)

	// Static file server (serves build/dev)
	mux.Handle("/", http.FileServer(http.Dir(dir)))

	log.Printf("dev server: http://localhost%s", addr)
	return http.ListenAndServe(addr, mux)
}
