package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/alias-asso/iosu/internal/repository"
	"github.com/alias-asso/iosu/internal/service"
)

// route handler
func (s *Server) postCreateContest(w http.ResponseWriter, r *http.Request) {
	layout := "2006-01-02T15:04"

	startTime, err := time.Parse(layout, r.FormValue("startTime"))
	if err != nil {
		http.Error(w, "Invalid start time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(layout, r.FormValue("endTime"))
	if err != nil {
		http.Error(w, "Invalid end time format", http.StatusBadRequest)
		return
	}

	input := service.CreateContestInput{
		Name:      r.FormValue("name"),
		StartTime: startTime,
		EndTime:   endTime,
	}

	err = s.contestService.CreateContest(r.Context(), input)
	if err != nil {
		switch err {
		case service.ErrNameTooLong,
			service.ErrContestAlreadyExists,
			service.ErrDirectoryExists,
			service.ErrInvalidTimeRange:
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) patchUpdateContest(w http.ResponseWriter, r *http.Request) {

	idValue := r.FormValue("id")

	id64, err := strconv.ParseUint(idValue, 10, 64)
	if err != nil {
		http.Error(w, "invalid contest id", http.StatusBadRequest)
		return
	}

	var input service.UpdateContestInput
	input.ID = uint(id64)

	if v := r.FormValue("name"); v != "" {
		input.Name = &v
	}

	layout := "2006-01-02T15:04"

	if v := r.FormValue("startTime"); v != "" {
		t, err := time.Parse(layout, v)
		if err != nil {
			http.Error(w, "invalid start time format", http.StatusBadRequest)
			return
		}
		input.StartTime = &t
	}

	if v := r.FormValue("endTime"); v != "" {
		t, err := time.Parse(layout, v)
		if err != nil {
			http.Error(w, "invalid end time format", http.StatusBadRequest)
			return
		}
		input.EndTime = &t
	}

	err = s.contestService.UpdateContest(r.Context(), input)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrNameTooLong),
			errors.Is(err, service.ErrInvalidTimeRange):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, repository.ErrContestNotFound):
			http.Error(w, "contest not found", http.StatusNotFound)

		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getContest(w http.ResponseWriter, r *http.Request) {
	contestName := r.PathValue("name")
	getProblemsInput := service.GetProblemsInput{
		ContestName: contestName,
	}
	problems, err := s.problemService.GetProblems(r.Context(), getProblemsInput)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrContestNotFound):
			s.render(w, "pages/error.gohtml", err.Error())
			return
		default:
			s.render(w, "pages/error.gohtml", "internal server error")
			return
		}
	}
	s.render(w, "pages/contest.gohtml", problems)
	return
}
