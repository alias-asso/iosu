// Package web serves the site: routing, session handling and HTML rendering.
package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/config"
)

//go:embed static views
var content embed.FS

type Server struct {
	app       *app.App
	cfg       *config.Config
	mux       *http.ServeMux
	templates map[string]*template.Template
}

// New builds the server and parses every template up front, so a broken
// template fails at startup rather than in the middle of a request.
func New(a *app.App, cfg *config.Config) (*Server, error) {
	templates, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	s := &Server{app: a, cfg: cfg, mux: http.NewServeMux(), templates: templates}
	s.routes()
	return s, nil
}

// Handler returns the fully wrapped handler: panics recovered, security
// headers set, request bodies capped.
func (s *Server) Handler() http.Handler {
	return recoverPanic(securityHeaders(limitBody(s.mux)))
}

// Start listens until the returned server is shut down.
func (s *Server) Start(port string) error {
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 16,
	}
	log.Printf("listening on http://localhost:%s", port)
	return srv.ListenAndServe()
}

// parseTemplates builds one template set per page, each combining the layout
// and the shared partials.
func parseTemplates() (map[string]*template.Template, error) {
	pages, err := fs.Glob(content, "views/pages/*.gohtml")
	if err != nil {
		return nil, err
	}
	shared := []string{
		"views/layout/base.gohtml",
		"views/partials/header.gohtml",
		"views/partials/footer.gohtml",
	}

	out := make(map[string]*template.Template, len(pages)+1)
	for _, page := range pages {
		files := append(append([]string{}, shared...), page)
		tpl, err := template.ParseFS(content, files...)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", page, err)
		}
		out[name(page)] = tpl
	}

	partials, err := fs.Glob(content, "views/partials/*.gohtml")
	if err != nil {
		return nil, err
	}
	for _, p := range partials {
		tpl, err := template.ParseFS(content, p)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}
		out[name(p)] = tpl
	}
	return out, nil
}

// name turns "views/pages/index.gohtml" into "index".
func name(path string) string {
	base := path[len("views/"):]
	return base[:len(base)-len(".gohtml")]
}
