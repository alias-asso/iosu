package server

import (
	"net/http"
	"time"

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

	err = s.ContestService.CreateContest(r.Context(), input)
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
