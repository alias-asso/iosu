package server

import (
	"errors"
	"net/http"

	"github.com/alias-asso/iosu/internal/service"
)

func (s *Server) getProblem(w http.ResponseWriter, r *http.Request) {
	contestName := r.PathValue("name")
	problemSlug := r.PathValue("slug")

	getProblemInput := service.GetProblemInput{
		Slug: problemSlug,
	}
	problem, err := s.problemService.GetProblem(r.Context(), getProblemInput)
	if err != nil {
		s.render(w, "pages/error.gohtml", err.Error())
	}

	if problem.Contest.Name != contestName {
		s.render(w, "pages/error.gohtml", errors.New("Le problème ne fait pas partie du tournois."))
	}

	ctx := r.Context()
	claims := ctx.Value("claims").(*service.Claims)

	getProblemPartsHtmlInput := service.GetProblemPartHtmlInput{
		Slug:   problemSlug,
		UserID: claims.UserID,
	}

	parts, err := s.problemService.GetProblemPartsHtml(r.Context(), getProblemPartsHtmlInput)
	if err != nil {
		s.render(w, "pages/error.gohtml", service.ErrPartNotFound.Error())
	}

	s.render(w, "pages/problem.gohtml", struct {
		Title   string
		Content []string
	}{
		Title:   problem.Name,
		Content: parts})
}
