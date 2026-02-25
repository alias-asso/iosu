package server

import (
	"net/http"
	"time"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
)

// route handler
func (s *Server) postCreateContest(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	startTimeValue := r.FormValue("startTime")
	endTimeValue := r.FormValue("endTime")

	if len(name) >= 20 {
		http.Error(w, "Name is too long", http.StatusBadRequest)
		return
	}

	layout := "2006-01-02T15:04"
	startTime, err := time.Parse(layout, startTimeValue)
	if err != nil {
		http.Error(w, "Invalid start time date format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(layout, endTimeValue)
	if err != nil {
		http.Error(w, "Invalid end time date format", http.StatusBadRequest)
		return
	}

	contest := database.Contest{
		Name:      name,
		StartTime: startTime,
		EndTime:   endTime,
	}

	err = gorm.G[database.Contest](s.db).Create(r.Context(), &contest)
	if err != nil {
		http.Error(w, "Error inserting contest", http.StatusInternalServerError)
	}

	contestDirPath :=

		// TODO: change
		w.Write([]byte("inserted successfully"))
}
