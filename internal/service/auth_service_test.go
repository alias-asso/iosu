package service

import (
	"context"
	"errors"
	"testing"

	"github.com/alias-asso/iosu/internal/database"
)

type mockUserRepo struct {
	getByUsernameFn             func(ctx context.Context, username string) (database.User, error)
	createUserWithActivationFn  func(ctx context.Context, user *database.User, activation *database.ActivationCode) error
	createIfNotExistFn          func(ctx context.Context, user *database.User) (bool, error)
	updateByUsernameFn          func(ctx context.Context, user database.User) error
	getFn                       func(ctx context.Context, userID uint) (database.User, error)
	getActivationCodeFn         func(ctx context.Context, code string) (database.ActivationCode, error)
	getActivationCodesFn        func(ctx context.Context) ([]database.ActivationCode, error)
	getNonAdminUsersWithSolveFn func(ctx context.Context) ([]database.User, error)
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (database.User, error) {
	return m.getByUsernameFn(ctx, username)
}

func (m *mockUserRepo) CreateUserWithActivation(ctx context.Context, user *database.User, activation *database.ActivationCode) error {
	return m.createUserWithActivationFn(ctx, user, activation)
}

func (m *mockUserRepo) CreateIfNotExist(ctx context.Context, user *database.User) (bool, error) {
	return m.createIfNotExistFn(ctx, user)
}

func (m *mockUserRepo) UpdateByUsername(ctx context.Context, user database.User) error {
	return m.updateByUsernameFn(ctx, user)
}

func (m *mockUserRepo) Get(ctx context.Context, userID uint) (database.User, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userID)
	}
	return database.User{}, nil
}

func (m *mockUserRepo) GetActivationCode(ctx context.Context, code string) (database.ActivationCode, error) {
	return m.getActivationCodeFn(ctx, code)
}

func (m *mockUserRepo) GetActivationCodes(ctx context.Context) ([]database.ActivationCode, error) {
	return m.getActivationCodesFn(ctx)
}

func (m *mockUserRepo) GetNonAdminUsersWithSolves(ctx context.Context) ([]database.User, error) {
	return m.getNonAdminUsersWithSolveFn(ctx)
}

func TestLogin(t *testing.T) {

	hash, _ := encryptPassword("password")

	tests := []struct {
		name    string
		input   LoginInput
		user    database.User
		repoErr error
		wantErr error
		wantJWT bool
	}{
		{
			name: "ok",
			input: LoginInput{
				Username: "user",
				Password: "password",
			},
			user: database.User{
				Username: "user",
				Password: hash,
			},
			wantJWT: true,
		},
		{
			name: "username required",
			input: LoginInput{
				Username: "",
				Password: "pass",
			},
			wantErr: ErrUsernameRequired,
		},
		{
			name: "password required",
			input: LoginInput{
				Username: "user",
				Password: "",
			},
			wantErr: ErrPasswordRequired,
		},
		{
			name: "account not found",
			input: LoginInput{
				Username: "user",
				Password: "password",
			},
			repoErr: errors.New("not found"),
			wantErr: ErrNonExistantAccount,
		},
		{
			name: "wrong password",
			input: LoginInput{
				Username: "user",
				Password: "wrong",
			},
			user: database.User{
				Username: "user",
				Password: hash,
			},
			wantErr: ErrWrongPassword,
		},
	}

	for _, tt := range tests {

		repo := &mockUserRepo{
			getByUsernameFn: func(ctx context.Context, username string) (database.User, error) {
				if tt.repoErr != nil {
					return database.User{}, tt.repoErr
				}
				return tt.user, nil
			},
		}

		service := NewAuthService(repo, "secret", "adminpass")

		token, err := service.Login(context.Background(), tt.input)

		if tt.wantErr == nil && err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
			t.Fatalf("expected %v got %v", tt.wantErr, err)
		}

		if tt.wantJWT && token == "" {
			t.Fatalf("expected jwt token")
		}
	}
}

func TestRegister(t *testing.T) {

	tests := []struct {
		name    string
		input   RegisterInput
		repoErr error
		wantErr error
	}{
		{
			name: "ok",
			input: RegisterInput{
				Username: "user",
				Email:    "user@test.com",
				Password: "password",
			},
		},
		{
			name: "username required",
			input: RegisterInput{
				Username: "",
				Email:    "user@test.com",
				Password: "password",
			},
			wantErr: ErrUsernameRequired,
		},
		{
			name: "password required",
			input: RegisterInput{
				Username: "user",
				Email:    "user@test.com",
				Password: "",
			},
			wantErr: ErrPasswordRequired,
		},
		{
			name: "invalid email",
			input: RegisterInput{
				Username: "user",
				Email:    "notanemail",
				Password: "password",
			},
			wantErr: ErrInvalidEmail,
		},
		{
			name: "username too long",
			input: RegisterInput{
				Username: "thisusernameistoolong",
				Email:    "user@test.com",
				Password: "password",
			},
			wantErr: ErrUsernameTooLong,
		},
	}

	for _, tt := range tests {

		repo := &mockUserRepo{
			createUserWithActivationFn: func(ctx context.Context, user *database.User, activation *database.ActivationCode) error {
				return tt.repoErr
			},
		}

		service := NewAuthService(repo, "secret", "adminpass")

		err := service.Register(context.Background(), tt.input)

		if tt.wantErr == nil && err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
			t.Fatalf("expected %v got %v", tt.wantErr, err)
		}
	}
}

func TestBatchRegister(t *testing.T) {

	validCSV := "username,email\nuser1,user1@test.com\nuser2,user2@test.com"

	tests := []struct {
		name    string
		csv     string
		repoErr error
		wantErr error
	}{
		{
			name: "ok",
			csv:  validCSV,
		},
		{
			name:    "invalid csv",
			csv:     "",
			wantErr: ErrInvalidCSV,
		},
		{
			name:    "invalid header",
			csv:     "name,mail\nuser,test@test.com",
			wantErr: ErrInvalidCSVHeader,
		},
		{
			name:    "invalid email",
			csv:     "username,email\nuser,notanemail",
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tt := range tests {

		repo := &mockUserRepo{
			createUserWithActivationFn: func(ctx context.Context, user *database.User, activation *database.ActivationCode) error {
				return tt.repoErr
			},
		}

		service := NewAuthService(repo, "secret", "adminpass")

		err := service.BatchRegister(context.Background(), tt.csv)

		if tt.wantErr == nil && err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
			t.Fatalf("expected %v got %v", tt.wantErr, err)
		}
	}
}
