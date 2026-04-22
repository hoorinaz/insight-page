package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"

	"github.com/yourusername/page-insight-tool/internal/handler"
)

//go:embed static
var staticFiles embed.FS

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	// Serve files rooted at the static/ subdirectory.
	subFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("failed to set up static filesystem: %v", err)
	}

	srv := &http.Server{
		Addr:    *addr,
		Handler: handler.New(http.FS(subFS)),
	}

	log.Printf("Page Insight Tool listening on http://localhost%s", *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
