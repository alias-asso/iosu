package web

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/alias-asso/iosu/internal/app"
)

type indexPage struct {
	Contest *app.Contest       // nil when no contest is active
	Archive []app.ArchiveEntry // only loaded when Contest is nil
}

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) {
	config, err := s.app.SiteConfig(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	var page indexPage
	if slug := config.CurrentContest; slug != "" {
		contest, err := s.app.Contest(r.Context(), slug)
		if err != nil {
			// A stale slug must not take the home page down; fall back to
			// the archive as though no contest were active.
			log.Printf("current contest %q: %v", slug, err)
		} else {
			page.Contest = &contest
		}
	}
	if page.Contest == nil {
		if page.Archive, err = s.app.Archive(r.Context()); err != nil {
			s.renderError(w, r, err)
			return
		}
	}
	s.renderWith(w, r, "index", page, config)
}

func (s *Server) getArchive(w http.ResponseWriter, r *http.Request) {
	entries, err := s.app.Archive(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "archive", entries)
}

func (s *Server) markdownPage(pick func(app.SiteConfig) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config, err := s.app.SiteConfig(r.Context())
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		html, err := app.Markdown(pick(config))
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		s.renderWith(w, r, "page", html, config)
	}
}

type loginPage struct{ Error string }

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFrom(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, r, "login", loginPage{})
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	user, err := s.app.Authenticate(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		msg, status := describe(err)
		w.WriteHeader(status)
		s.render(w, r, "login", loginPage{Error: msg})
		return
	}

	token, err := s.issueToken(user)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.setTokenCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	s.clearTokenCookie(w)
	w.Header().Set("HX-Redirect", "/")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type activatePage struct {
	Error          string
	ActivationCode string
}

func (s *Server) getActivate(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if _, err := s.app.ActivationCode(r.Context(), code); err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "activate-account", activatePage{ActivationCode: code})
}

func (s *Server) postActivate(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("activation-code")
	err := s.app.Activate(r.Context(), code, r.FormValue("password"))
	if err != nil {
		msg, status := describe(err)
		w.WriteHeader(status)
		s.render(w, r, "activate-account", activatePage{Error: msg, ActivationCode: code})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

type contestPage struct {
	Contest  app.Contest
	Problems []app.ProblemInList
}

func (s *Server) getContest(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	contest, err := s.app.Contest(r.Context(), slug)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	problems, err := s.app.Problems(r.Context(), slug)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "contest", contestPage{Contest: contest, Problems: problems})
}

func (s *Server) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	board, err := s.app.Leaderboard(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "leaderboard", board)
}

type problemPage struct {
	Problem     app.Problem
	Difficulty  app.Difficulty
	SolvedParts int64
	Content     []template.HTML
}

func (s *Server) getProblem(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)

	detail, err := s.app.ProblemIn(r.Context(), r.PathValue("contest"), r.PathValue("problem"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	solved, err := s.app.SolvedParts(r.Context(), user.ID, detail.Problem.ID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	parts, err := s.app.ProblemStatement(r.Context(), user.ID, detail)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	s.render(w, r, "problem", problemPage{
		Problem:     detail.Problem,
		Difficulty:  detail.Difficulty,
		SolvedParts: solved,
		Content:     parts,
	})
}

func (s *Server) getInput(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)

	detail, err := s.app.ProblemIn(r.Context(), r.PathValue("contest"), r.PathValue("problem"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	input, err := s.app.ProblemInput(r.Context(), user.ID, detail)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\""+detail.Problem.Slug+".txt\"")
	if _, err := w.Write([]byte(input)); err != nil {
		log.Printf("writing input for %s: %v", detail.Problem.Slug, err)
	}
}

func (s *Server) getProblemImage(w http.ResponseWriter, r *http.Request) {
	path, err := s.app.ProblemImage(r.PathValue("contest"), r.PathValue("problem"), r.PathValue("img"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

type responseIndicator struct {
	Error   string
	Success bool
	MaxPart bool
}

func (s *Server) postSubmit(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)

	part, err := strconv.ParseInt(r.PathValue("part"), 10, 64)
	if err != nil {
		s.renderPartial(w, "response-indicator", responseIndicator{Error: "Partie invalide."})
		return
	}

	in := app.SubmitInput{
		UserID:      user.ID,
		ContestSlug: r.PathValue("contest"),
		ProblemSlug: r.PathValue("problem"),
		Part:        part,
		Answer:      r.FormValue("response"),
	}

	correct, err := s.app.Submit(r.Context(), in)
	if err != nil {
		msg, status := describe(err)
		if status == http.StatusInternalServerError {
			log.Printf("submit %s part %d: %v", in.ProblemSlug, part, err)
		}
		s.renderPartial(w, "response-indicator", responseIndicator{Error: msg})
		return
	}
	if !correct {
		s.renderPartial(w, "response-indicator", responseIndicator{})
		return
	}

	detail, err := s.app.Problem(r.Context(), in.ProblemSlug)
	if err != nil {
		// The answer was accepted; only the "is this the last part?" hint is lost.
		s.renderPartial(w, "response-indicator", responseIndicator{Success: true})
		return
	}
	s.renderPartial(w, "response-indicator", responseIndicator{
		Success: true,
		MaxPart: part == detail.Problem.Parts,
	})
}

func (s *Server) getAdminConfig(w http.ResponseWriter, r *http.Request) {
}
