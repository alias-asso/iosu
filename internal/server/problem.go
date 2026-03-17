package server

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/alias-asso/iosu/internal/database"
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
		s.render(w, r.Context(), "pages/error.gohtml", err.Error())
		return
	}

	if problem.Contest.Name != contestName {
		s.render(w, r.Context(), "pages/error.gohtml", errors.New("Le problème ne fait pas partie du tournois."))
		return
	}

	ctx := r.Context()
	claims := ctx.Value("claims").(*service.Claims)

	getProblemPartsHtmlInput := service.GetProblemPartHtmlInput{
		Slug:   problemSlug,
		UserID: claims.UserID,
	}

	parts, err := s.problemService.GetProblemPartsHtml(r.Context(), getProblemPartsHtmlInput)
	if err != nil {
		s.render(w, r.Context(), "pages/error.gohtml", service.ErrPartNotFound.Error())
		return
	}

	s.render(w, r.Context(), "pages/problem.gohtml", struct {
		Problem database.Problem
		Content []template.HTML
	}{
		Problem: problem,
		Content: parts})
}

func (s *Server) getProblemImages(w http.ResponseWriter, r *http.Request) {
	contestName := r.PathValue("name")
	problemSlug := r.PathValue("slug")
	imageName := r.PathValue("img")

	if strings.Contains(imageName, "/") || strings.Contains(imageName, "\\") {
		http.Error(w, "invalid image name", http.StatusBadRequest)
		return
	}

	baseDir := filepath.Join(s.cfg.DataDirectory, contestName, problemSlug, "img")

	fullPath := filepath.Join(baseDir, imageName)
	fullPath = filepath.Clean(fullPath)

	baseDirClean := filepath.Clean(baseDir) + string(os.PathSeparator)
	if !strings.HasPrefix(fullPath, baseDirClean) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, fullPath)
}
