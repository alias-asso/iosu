package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestServer(t *testing.T) *Server {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	err = db.AutoMigrate(&database.Contest{})
	if err != nil {
		t.Fatal(err)
	}

	return &Server{
		db: db,
		cfg: &config.Config{
			DataDirectory: t.TempDir(),
		},
	}
}

func TestCreateContest(t *testing.T) {
	server := newTestServer(t)
	t.Run("NewContest", func(t *testing.T) {
		form := url.Values{}
		form.Set("name", "contest1")
		form.Set("startTime", "2025-01-01T10:00")
		form.Set("endTime", "2025-01-01T12:00")

		req := httptest.NewRequest(http.MethodPost, "/contest",
			strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		w := httptest.NewRecorder()

		server.postCreateContest(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 got %d", w.Code)
		}

		// Verify DB actually contains the contest
		var count int64
		server.db.Model(&database.Contest{}).Count(&count)

		if count != 1 {
			t.Fatalf("expected 1 contest, got %d", count)
		}
	})
}
