package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"
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

func parseTemplates() (map[string]*template.Template, error) {
	partials, err := fs.Glob(content, "views/partials/*.gohtml")
	if err != nil {
		return nil, err
	}

	out := make(map[string]*template.Template)
	for _, l := range []struct{ pages, layout string }{
		{"views/pages/*.gohtml", "views/layout/base.gohtml"},
		{"views/pages/admin/*.gohtml", "views/layout/admin.gohtml"},
	} {
		pages, err := fs.Glob(content, l.pages)
		if err != nil {
			return nil, err
		}
		for _, page := range pages {
			files := append(append([]string{}, partials...), l.layout, page)
			tpl, err := template.New(path.Base(files[0])).Funcs(template.FuncMap{
				"salmond": salmondText,
			}).ParseFS(content, files...)
			if err != nil {
				return nil, fmt.Errorf("parsing %s: %w", page, err)
			}
			out[name(page)] = tpl
		}
	}

	for _, p := range partials {
		tpl, err := template.New(path.Base(p)).Funcs(template.FuncMap{
			"salmond": salmondText,
		}).ParseFS(content, p)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", p, err)
		}
		out[name(p)] = tpl
	}
	return out, nil
}

var salmondReplacer = strings.NewReplacer(
	"À", "A", "Á", "A", "Â", "A", "Ä", "A", "à", "a", "á", "a", "â", "a", "ä", "a",
	"Æ", "AE", "æ", "ae", "Ç", "C", "ç", "c",
	"È", "E", "É", "E", "Ê", "E", "Ë", "E", "è", "e", "é", "e", "ê", "e", "ë", "e",
	"Î", "I", "Ï", "I", "î", "i", "ï", "i", "Ô", "O", "Ö", "O", "ô", "o", "ö", "o",
	"Œ", "OE", "œ", "oe", "Ù", "U", "Û", "U", "Ü", "U", "ù", "u", "û", "u", "ü", "u",
	"Ÿ", "Y", "ÿ", "y",
)

func salmondText(text string) string { return salmondReplacer.Replace(text) }

func name(path string) string {
	base := path[len("views/"):]
	return base[:len(base)-len(".gohtml")]
}
