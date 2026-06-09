package main

import (
	"net/http"
	"os"
	"path/filepath"
)

// mediaHandler serves local audio files to the webview's <audio> element at
// `GET /media?path=<urlencoded-abs-path>`. It is wired as the AssetServer
// fallback Handler, so it only sees requests the embedded frontend didn't
// satisfy.
//
// Using http.ServeContent gives HTTP Range support for free, which the
// <audio> element needs for seeking and which keeps memory flat (no full-file
// blob in the webview).
//
// Security: the path MUST be in the App's media allowlist (populated on scan),
// otherwise this would be an arbitrary-file-read primitive. We refuse anything
// not currently loaded.
func mediaHandler(app *App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/media" {
			http.NotFound(w, r)
			return
		}
		p := r.URL.Query().Get("path")
		if p == "" {
			http.Error(w, "missing path", http.StatusBadRequest)
			return
		}
		clean := filepath.Clean(p)
		if app.mediaSet == nil || !app.mediaSet[clean] {
			logger.Printf("media: refused (not in allowlist): %s", clean)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		f, err := os.Open(clean)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			http.Error(w, "stat failed", http.StatusInternalServerError)
			return
		}
		// ServeContent picks Content-Type from the name's extension and handles
		// Range requests + seeking.
		http.ServeContent(w, r, st.Name(), st.ModTime(), f)
	})
}
