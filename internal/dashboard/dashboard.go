// internal/dashboard/dashboard.go
package dashboard

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"

	"load-balancer/internal/server"
)

//go:embed templates static
var content embed.FS

// Handler serves the dashboard UI
func Handler(srvMgr *server.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Dashboard handling request: %s", r.URL.Path)

		// Serve static files
		if strings.HasPrefix(r.URL.Path, "/static/") {
			// Log the requested file path
			filePath := strings.TrimPrefix(r.URL.Path, "/")
			log.Printf("Attempting to serve static file: %s", filePath)

			// Try to open the file
			file, err := content.Open(filePath)
			if err != nil {
				log.Printf("Error opening static file: %v", err)
				http.NotFound(w, r)
				return
			}
			defer file.Close()

			// Set content type based on file extension
			contentType := "application/octet-stream"
			switch path.Ext(r.URL.Path) {
			case ".css":
				contentType = "text/css"
			case ".js":
				contentType = "application/javascript"
			case ".svg":
				contentType = "image/svg+xml"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			}
			w.Header().Set("Content-Type", contentType)

			// Serve the file
			http.FileServer(http.FS(content)).ServeHTTP(w, r)
			return
		}

		// Serve the dashboard HTML for root path or non-API paths
		if r.URL.Path == "/" || !strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("Serving dashboard HTML")
			tmpl, err := template.ParseFS(content, "templates/dashboard.html")
			if err != nil {
				log.Printf("Error parsing template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			err = tmpl.Execute(w, nil)
			if err != nil {
				log.Printf("Error executing template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		// If we get here, it's likely an API path that should be handled elsewhere
		http.NotFound(w, r)
	}
}
