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
	s.mux.HandleFunc("GET /archive", s.optionalAuth(s.getArchive))

	s.mux.HandleFunc("GET /login", s.optionalAuth(s.getLogin))
	s.mux.HandleFunc("POST /login", s.postLogin)
	s.mux.HandleFunc("POST /logout", s.postLogout)
	s.mux.HandleFunc("GET /register", s.optionalAuth(s.getRegister))
	s.mux.HandleFunc("POST /register", s.optionalAuth(s.postRegister))

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

	// Admin pages
	s.mux.HandleFunc("GET /admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/config", http.StatusFound)
	})
	s.mux.HandleFunc("GET /admin/config", s.requireAdmin(s.getAdminConfig))
	s.mux.HandleFunc("POST /admin/config", s.requireAdmin(s.postAdminConfig))
	s.mux.HandleFunc("GET /admin/users", s.requireAdmin(s.getAdminUsers))
	s.mux.HandleFunc("POST /admin/users/import", s.requireAdmin(s.postAdminUsersImport))
	s.mux.HandleFunc("GET /admin/users/new", s.requireAdmin(s.getAdminUserNew))
	s.mux.HandleFunc("POST /admin/users/new", s.requireAdmin(s.postAdminUserNew))
	s.mux.HandleFunc("GET /admin/users/{user}/edit", s.requireAdmin(s.getAdminUserEdit))
	s.mux.HandleFunc("POST /admin/users/{user}/edit", s.requireAdmin(s.postAdminUserEdit))
	s.mux.HandleFunc("GET /admin/users/{user}/admin", s.requireAdmin(s.getAdminUserAdmin))
	s.mux.HandleFunc("POST /admin/users/{user}/admin", s.requireAdmin(s.postAdminUserAdmin))
	s.mux.HandleFunc("POST /admin/users/{user}/approve", s.requireAdmin(s.postAdminUserApprove))
	s.mux.HandleFunc("GET /admin/users/{user}/delete", s.requireAdmin(s.getAdminUserDelete))
	s.mux.HandleFunc("POST /admin/users/{user}/delete", s.requireAdmin(s.postAdminUserDelete))
	s.mux.HandleFunc("GET /admin/contests", s.requireAdmin(s.getAdminContests))
	s.mux.HandleFunc("GET /admin/contests/new", s.requireAdmin(s.getAdminContestNew))
	s.mux.HandleFunc("POST /admin/contests/new", s.requireAdmin(s.postAdminContestNew))
	s.mux.HandleFunc("GET /admin/contests/{contest}/edit", s.requireAdmin(s.getAdminContestEdit))
	s.mux.HandleFunc("POST /admin/contests/{contest}/edit", s.requireAdmin(s.postAdminContestEdit))
	s.mux.HandleFunc("GET /admin/contests/{contest}/delete", s.requireAdmin(s.getAdminContestDelete))
	s.mux.HandleFunc("POST /admin/contests/{contest}/delete", s.requireAdmin(s.postAdminContestDelete))
	s.mux.HandleFunc("GET /admin/problems", s.requireAdmin(s.getAdminProblems))
	s.mux.HandleFunc("GET /admin/problems/new", s.requireAdmin(s.getAdminProblemNew))
	s.mux.HandleFunc("POST /admin/problems/new", s.requireAdmin(s.postAdminProblemNew))
	s.mux.HandleFunc("GET /admin/problems/{problem}/edit", s.requireAdmin(s.getAdminProblemEdit))
	s.mux.HandleFunc("POST /admin/problems/{problem}/edit", s.requireAdmin(s.postAdminProblemEdit))
	s.mux.HandleFunc("GET /admin/problems/{problem}/delete", s.requireAdmin(s.getAdminProblemDelete))
	s.mux.HandleFunc("POST /admin/problems/{problem}/delete", s.requireAdmin(s.postAdminProblemDelete))
	s.mux.HandleFunc("GET /admin/difficulties", s.requireAdmin(s.getAdminDifficultySelector))
	s.mux.HandleFunc("POST /admin/difficulties", s.requireAdmin(s.postAdminDifficulty))
}
