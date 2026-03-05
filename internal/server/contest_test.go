package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alias-asso/iosu/internal/service"
)

type mockContestService struct {
	createFn func(ctx context.Context, input service.CreateContestInput) error
	updateFn func(ctx context.Context, input service.UpdateContestInput) error
}

func (m *mockContestService) CreateContest(ctx context.Context, input service.CreateContestInput) error {
	return m.createFn(ctx, input)
}

func (m *mockContestService) UpdateContest(ctx context.Context, input service.UpdateContestInput) error {
	return m.updateFn(ctx, input)
}

func TestPostCreateContest(t *testing.T) {

	service := &mockContestService{
		createFn: func(ctx context.Context, input service.CreateContestInput) error {
			return nil
		},
	}

	server := &Server{
		contestService: service,
	}

	form := url.Values{}
	form.Set("name", "contest")
	form.Set("startTime", "2025-01-01T10:00")
	form.Set("endTime", "2025-01-01T12:00")

	req := httptest.NewRequest(http.MethodPost, "/contest", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	server.postCreateContest(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", w.Code)
	}
}

func TestPatchUpdateContest(t *testing.T) {

	service := &mockContestService{
		updateFn: func(ctx context.Context, input service.UpdateContestInput) error {
			return nil
		},
	}

	server := &Server{
		contestService: service,
	}

	form := url.Values{}
	form.Set("id", "1")
	form.Set("name", "updated")

	req := httptest.NewRequest(http.MethodPatch, "/contest", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	server.patchUpdateContest(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", w.Code)
	}
}

func TestPatchUpdateContestInvalidID(t *testing.T) {

	server := &Server{}

	form := url.Values{}
	form.Set("id", "abc")

	req := httptest.NewRequest(http.MethodPatch, "/contest", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	server.patchUpdateContest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}
