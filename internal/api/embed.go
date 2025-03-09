package api

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

//go:embed templates/* static/*
var embeddedFiles embed.FS

// GetTemplates returns a filesystem with just the templates
func GetTemplates() *template.Template {
	tmpl := template.New("")
	template.Must(tmpl.ParseFS(embeddedFiles, "templates/*.html"))
	return tmpl
}

// ServeStaticFiles sets up the static file server handler using embedded files
func ServeStaticFiles(router *mux.Router) {
	staticFS, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create sub-filesystem for static files: %v", err)
	}

	// Debug: List files in the embedded filesystem
	log.Println("Listing embedded static files:")
	fs.WalkDir(staticFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		log.Printf("  - %s (%v)", path, d.IsDir())
		return nil
	})

	// Create a file server handler for the static files
	fileServer := http.FileServer(http.FS(staticFS))

	// Register the handler for the /static/ path
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", fileServer),
	)
}
