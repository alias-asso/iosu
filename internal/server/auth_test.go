package server

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alias-asso/iosu/internal/service"
)

type mockAuthService struct {
	loginFunc         func(ctx context.Context, input service.LoginInput) (string, error)
	batchRegisterFunc func(ctx context.Context, csv string) error
}

func (m *mockAuthService) Login(ctx context.Context, input service.LoginInput) (string, error) {
	return m.loginFunc(ctx, input)
}

func (m *mockAuthService) BatchRegister(ctx context.Context, csv string) error {
	return m.batchRegisterFunc(ctx, csv)
}

func (m *mockAuthService) Register(ctx context.Context, input service.RegisterInput) error {
	return nil
}

func (m *mockAuthService) CreateDefaultAdmin(ctx context.Context) {}

func TestPostLogin_Success(t *testing.T) {
	mock := &mockAuthService{
		loginFunc: func(ctx context.Context, input service.LoginInput) (string, error) {
			if input.Username != "admin" || input.Password != "password" {
				t.Fatalf("unexpected input")
			}
			return "jwt-token", nil
		},
	}

	s := &Server{
		authService: mock,
	}

	form := bytes.NewBufferString("username=admin&password=password")

	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	s.postLogin(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 got %d", res.StatusCode)
	}

	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie to be set")
	}

	if cookies[0].Name != "token" {
		t.Fatalf("expected token cookie got %s", cookies[0].Name)
	}
}

func TestPostLogin_Error(t *testing.T) {
	mock := &mockAuthService{
		loginFunc: func(ctx context.Context, input service.LoginInput) (string, error) {
			return "", errors.New("invalid credentials")
		},
	}

	s := &Server{
		authService: mock,
	}

	form := bytes.NewBufferString("username=test&password=test")

	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()

	s.postLogin(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", res.StatusCode)
	}
}

func TestPostBatchCreateAccounts_Success(t *testing.T) {
	mock := &mockAuthService{
		batchRegisterFunc: func(ctx context.Context, csv string) error {
			return nil
		},
	}

	s := &Server{
		authService: mock,
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("accounts", "accounts.csv")
	part.Write([]byte("username,email\nuser1,test@example.com"))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/batch", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()

	s.postBatchCreateAccounts(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 got %d", res.StatusCode)
	}
}

func TestPostBatchCreateAccounts_InvalidCSV(t *testing.T) {
	mock := &mockAuthService{
		batchRegisterFunc: func(ctx context.Context, csv string) error {
			return service.ErrInvalidCSV
		},
	}

	s := &Server{
		authService: mock,
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("accounts", "accounts.csv")
	part.Write([]byte("bad csv"))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/batch", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()

	s.postBatchCreateAccounts(w, req)

	res := w.Result()

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", res.StatusCode)
	}
}
