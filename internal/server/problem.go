package server

import (
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/service"
)

func (s *Server) getProblem(w http.ResponseWriter, r *http.Request) {
	contestSlug := r.PathValue("contest_slug")
	problemSlug := r.PathValue("problem_slug")

	getProblemInput := service.GetProblemInput{
		Slug: problemSlug,
	}
	problem, err := s.problemService.GetProblem(r.Context(), getProblemInput)
	if err != nil {
		s.render(w, r.Context(), "pages/error.gohtml", err.Error())
		return
	}

	if problem.Contest.Name != contestSlug {
		s.render(w, r.Context(), "pages/error.gohtml", errors.New("Le problème ne fait pas partie du tournois."))
		return
	}

	ctx := r.Context()
	claims := ctx.Value("claims").(*service.Claims)
	getSolvesInput := service.GetSolvesInput{
		UserID:    claims.UserID,
		ProblemID: problem.ID,
	}

	solves, err := s.problemService.GetSolves(ctx, getSolvesInput)
	if err != nil {
		s.render(w, r.Context(), "pages/error.gohtml", service.ErrInternalError.Error())
		return
	}

	getProblemPartsHtmlInput := service.GetProblemPartHtmlInput{
		Slug:   problemSlug,
		UserID: claims.UserID,
	}

	parts, err := s.problemService.GetProblemPartsHtml(r.Context(), getProblemPartsHtmlInput)
	if err != nil {
		s.render(w, r.Context(), "pages/error.gohtml", err.Error())
		return
	}

	s.render(w, r.Context(), "pages/problem.gohtml", struct {
		SolvedParts uint
		Problem     database.Problem
		Content     []template.HTML
	}{
		SolvedParts: solves,
		Problem:     problem,
		Content:     parts})
}

func (s *Server) getProblemImages(w http.ResponseWriter, r *http.Request) {
	contestSlug := r.PathValue("contest_slug")
	problemSlug := r.PathValue("problem_slug")
	imageName := r.PathValue("img")

	if strings.Contains(imageName, "/") || strings.Contains(imageName, "\\") {
		http.Error(w, "invalid image name", http.StatusBadRequest)
		return
	}

	baseDir := filepath.Join(s.cfg.DataDirectory, contestSlug, problemSlug, "img")

	fullPath := filepath.Join(baseDir, imageName)
	fullPath = filepath.Clean(fullPath)

	baseDirClean := filepath.Clean(baseDir) + string(os.PathSeparator)
	if !strings.HasPrefix(fullPath, baseDirClean) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func (s *Server) postSubmit(w http.ResponseWriter, r *http.Request) {
	contestSlug := r.PathValue("contest_slug")
	problemSlug := r.PathValue("problem_slug")
	partString := r.PathValue("part")
	part, err := strconv.ParseUint(partString, 10, 32)
	if err != nil {
		http.Error(w, "invalid part", http.StatusBadRequest)
		return
	}

	value := r.FormValue("response")

	ctx := r.Context()
	claims := ctx.Value("claims").(*service.Claims)

	input := service.SubmitInput{
		UserID:      claims.UserID,
		Slug:        problemSlug,
		ContestSlug: contestSlug,
		Value:       value,
		Part:        uint(part),
	}
	ok, err := s.problemService.Submit(ctx, input)
	type Data struct {
		Error   string
		Success bool
		MaxPart bool
	}
	if ok {
		gpInput := service.GetProblemInput{
			Slug: problemSlug,
		}
		problem, err := s.problemService.GetProblem(ctx, gpInput)
		if err != nil {
			s.renderPartial(w, ctx, "partials/response-indicator.gohtml", Data{Error: err.Error(), Success: false, MaxPart: false})
		}
		if problem.Parts == uint(part) {
			s.renderPartial(w, ctx, "partials/response-indicator.gohtml", Data{Error: "", Success: true, MaxPart: true})
			return
		}
		s.renderPartial(w, ctx, "partials/response-indicator.gohtml", Data{Error: "", Success: true, MaxPart: false})
		return
	} else {
		if err != nil {
			s.renderPartial(w, ctx, "partials/response-indicator.gohtml", Data{Error: err.Error(), Success: false, MaxPart: false})
			return
		}
		s.renderPartial(w, ctx, "partials/response-indicator.gohtml", Data{Error: "", Success: false})
		return
	}
}

func (s *Server) getInput(w http.ResponseWriter, r *http.Request) {
	contestSlug := r.PathValue("contest_slug")
	problemSlug := r.PathValue("problem_slug")

	ctx := r.Context()
	claims := ctx.Value("claims").(*service.Claims)

	getProblemInput := service.GetProblemInput{
		Slug: problemSlug,
	}
	problem, err := s.problemService.GetProblem(ctx, getProblemInput)
	if err != nil {
		http.Error(w, "this problem does not exist", http.StatusBadRequest)
		return
	}

	if problem.Contest.Slug != contestSlug {
		http.Error(w, "this problem is not part of this contest", http.StatusBadRequest)
		return
	}

	input := service.GetProblemInputInput{
		UserID:    claims.UserID,
		ProblemID: problem.ID,
	}

	value, err := s.problemService.GetProblemInput(ctx, input)
	if err != nil {
		http.Error(w, "unable to get input", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(value))
}
