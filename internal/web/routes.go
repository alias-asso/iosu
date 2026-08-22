package web

import (
	"io/fs"
	"net/http"

	"github.com/alias-asso/iosu/internal/app"
)

func (s *Server) routes() {
	static, _ := fs.Sub(content, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static", http.FileServer(http.FS(static))))

	s.mux.HandleFunc("GET /{$}", s.optionalAuth(s.getIndex))

	s.mux.HandleFunc("GET /login", s.optionalAuth(s.getLogin))
	s.mux.HandleFunc("POST /login", s.postLogin)
	s.mux.HandleFunc("POST /logout", s.postLogout)

	s.mux.HandleFunc("GET /activate/{code}", s.getActivate)
	s.mux.HandleFunc("POST /activate", s.postActivate)

	s.mux.HandleFunc("GET /contest/{slug}/", s.requireAuth(s.getContest))
	s.mux.HandleFunc("GET /contest/{slug}/leaderboard", s.requireAuth(s.getLeaderboard))

	s.mux.HandleFunc("GET /contest/{contest}/{problem}/", s.requireAuth(s.getProblem))
	s.mux.HandleFunc("GET /contest/{contest}/{problem}/input/", s.requireAuth(s.getInput))
	s.mux.HandleFunc("GET /contest/{contest}/{problem}/img/{img}", s.requireAuth(s.getProblemImage))
	s.mux.HandleFunc("POST /contest/{contest}/{problem}/submit/{part}", s.requireAuth(s.postSubmit))

	// The four static pages differ only in which markdown field they render.
	s.mux.HandleFunc("GET /help", s.optionalAuth(s.markdownPage(func(c app.SiteConfig) string { return c.HelpContent })))
	s.mux.HandleFunc("GET /rules", s.optionalAuth(s.markdownPage(func(c app.SiteConfig) string { return c.RulesContent })))
	s.mux.HandleFunc("GET /legal", s.optionalAuth(s.markdownPage(func(c app.SiteConfig) string { return c.LegalContent })))
	s.mux.HandleFunc("GET /credits", s.optionalAuth(s.markdownPage(func(c app.SiteConfig) string { return c.CreditsContent })))
}
