package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"slices"
	"strings"

	"github.com/alias-asso/iosu/internal/store/sqlc"
	"golang.org/x/crypto/bcrypt"
)

// dummyHash is compared against when the username is unknown, so that a login
// attempt costs the same whether or not the account exists.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy"), bcryptCost)

// Authenticate checks a username and password. Callers get ErrInvalidCredentials
// for both an unknown user and a wrong password so the response cannot be used
// to enumerate accounts.
func (a *App) Authenticate(ctx context.Context, username, password string) (User, error) {
	if username == "" || password == "" {
		return User{}, ErrInvalidCredentials
	}

	user, err := a.store.GetUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return User{}, ErrInvalidCredentials
	}
	if !user.Activated {
		return User{}, ErrNotActivated
	}
	return user, nil
}

// User looks up a user by ID.
func (a *App) User(ctx context.Context, id int64) (User, error) {
	user, err := a.store.GetUser(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

// UserByUsername looks up a user by name.
func (a *App) UserByUsername(ctx context.Context, username string) (User, error) {
	user, err := a.store.GetUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	return user, err
}

// EnsureAdmin creates the initial admin account, but only on a database with
// no users at all. It never touches an existing account, so the configured
// password cannot silently reset a changed one.
func (a *App) EnsureAdmin(ctx context.Context, password string) (created bool, err error) {
	n, err := a.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return false, err
	}
	_, err = a.store.CreateUser(ctx, sqlc.CreateUserParams{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: string(hash),
		Activated:    true,
		Admin:        true,
		CreatedAt:    a.now().Unix(),
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

// SetPassword replaces a user's password. This is how an admin password gets
// rotated.
func (a *App) SetPassword(ctx context.Context, username, password string) error {
	if !ValidPassword(password) {
		return ErrWeakPassword
	}
	user, err := a.UserByUsername(ctx, username)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}
	return a.store.SetUserPassword(ctx, sqlc.SetUserPasswordParams{
		PasswordHash: string(hash),
		ID:           user.ID,
	})
}

// SetAdmin grants or revokes admin rights.
func (a *App) SetAdmin(ctx context.Context, username string, admin bool) error {
	user, err := a.UserByUsername(ctx, username)
	if err != nil {
		return err
	}
	return a.store.SetUserAdmin(ctx, sqlc.SetUserAdminParams{Admin: admin, ID: user.ID})
}

// Register creates an unactivated account and returns its activation code.
func (a *App) Register(ctx context.Context, username, email string) (string, error) {
	if username == "" || len(username) > maxUsernameLen {
		return "", ErrInvalidUsername
	}
	if len(email) > maxEmailLen {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", ErrInvalidEmail
	}

	code, err := randomCode(activationCodeLen)
	if err != nil {
		return "", err
	}

	err = a.store.Tx(ctx, func(q *sqlc.Queries) error {
		user, err := q.CreateUser(ctx, sqlc.CreateUserParams{
			Username:  username,
			Email:     email,
			Activated: false,
			Admin:     false,
			CreatedAt: a.now().Unix(),
		})
		if err != nil {
			return err
		}
		return q.CreateActivationCode(ctx, sqlc.CreateActivationCodeParams{
			Code:      code,
			UserID:    user.ID,
			ExpiresAt: a.now().Add(activationTTL).Unix(),
		})
	})
	if err != nil {
		if isUniqueViolation(err) {
			return "", ErrUserExists
		}
		return "", err
	}
	return code, nil
}

// BatchRegister creates one account per row of a CSV with "username" and
// "email" columns. The whole file is applied or none of it is.
func (a *App) BatchRegister(ctx context.Context, csvContent string) (int, error) {
	reader := csv.NewReader(strings.NewReader(csvContent))
	header, err := reader.Read()
	if err != nil {
		return 0, ErrInvalidCSV
	}
	for i, h := range header {
		header[i] = strings.TrimSpace(strings.ToLower(h))
	}
	nameCol := slices.Index(header, "username")
	mailCol := slices.Index(header, "email")
	if nameCol < 0 || mailCol < 0 {
		return 0, fmt.Errorf("%w: need a header row with 'username' and 'email' columns", ErrInvalidCSV)
	}

	rows, err := reader.ReadAll()
	if err != nil {
		return 0, ErrInvalidCSV
	}

	type account struct{ username, email, code string }
	accounts := make([]account, 0, len(rows))
	for i, row := range rows {
		if nameCol >= len(row) || mailCol >= len(row) {
			return 0, fmt.Errorf("%w: row %d has too few columns", ErrInvalidCSV, i+2)
		}
		username := strings.TrimSpace(row[nameCol])
		email := strings.TrimSpace(row[mailCol])
		if username == "" || len(username) > maxUsernameLen {
			return 0, fmt.Errorf("row %d: %w", i+2, ErrInvalidUsername)
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return 0, fmt.Errorf("row %d: %w", i+2, ErrInvalidEmail)
		}
		code, err := randomCode(activationCodeLen)
		if err != nil {
			return 0, err
		}
		accounts = append(accounts, account{username, email, code})
	}

	err = a.store.Tx(ctx, func(q *sqlc.Queries) error {
		for _, acc := range accounts {
			user, err := q.CreateUser(ctx, sqlc.CreateUserParams{
				Username:  acc.username,
				Email:     acc.email,
				CreatedAt: a.now().Unix(),
			})
			if err != nil {
				if isUniqueViolation(err) {
					return fmt.Errorf("%s: %w", acc.username, ErrUserExists)
				}
				return err
			}
			if err := q.CreateActivationCode(ctx, sqlc.CreateActivationCodeParams{
				Code:      acc.code,
				UserID:    user.ID,
				ExpiresAt: a.now().Add(activationTTL).Unix(),
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(accounts), nil
}

// ActivationCode looks up a code and checks that it is still usable.
func (a *App) ActivationCode(ctx context.Context, code string) (sqlc.GetActivationCodeRow, error) {
	if len(code) != activationCodeLen {
		return sqlc.GetActivationCodeRow{}, ErrInvalidActivationCode
	}
	row, err := a.store.GetActivationCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return row, ErrInvalidActivationCode
	}
	if err != nil {
		return row, err
	}
	if row.ActivationCode.UsedAt.Valid {
		return row, ErrActivationCodeUsed
	}
	if a.now().Unix() > row.ActivationCode.ExpiresAt {
		return row, ErrActivationCodeExpired
	}
	return row, nil
}

// PendingActivations lists accounts that have not been activated yet.
func (a *App) PendingActivations(ctx context.Context) ([]PendingUser, error) {
	return a.store.ListPendingActivations(ctx)
}

// Activate consumes an activation code and sets the account's password. The
// code is marked used in the same transaction, so a link works exactly once.
func (a *App) Activate(ctx context.Context, code, password string) error {
	row, err := a.ActivationCode(ctx, code)
	if err != nil {
		return err
	}
	if !ValidPassword(password) {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}

	return a.store.Tx(ctx, func(q *sqlc.Queries) error {
		n, err := q.UseActivationCode(ctx, sqlc.UseActivationCodeParams{
			UsedAt: sql.NullInt64{Int64: a.now().Unix(), Valid: true},
			Code:   code,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrActivationCodeUsed
		}
		return q.ActivateUser(ctx, sqlc.ActivateUserParams{
			PasswordHash: string(hash),
			ID:           row.User.ID,
		})
	})
}

// Leaderboard returns the scores for one contest, highest first. Users with no
// points are left out.
func (a *App) Leaderboard(ctx context.Context, contestSlug string) ([]Leaderboarder, error) {
	contest, err := a.Contest(ctx, contestSlug)
	if err != nil {
		return nil, err
	}
	return a.store.Leaderboard(ctx, contest.ID)
}

var passwordChecks = []*regexp.Regexp{
	regexp.MustCompile(`[A-Z]`),
	regexp.MustCompile(`[a-z]`),
	regexp.MustCompile(`[0-9]`),
	regexp.MustCompile(`[^A-Za-z0-9]`),
}

// ValidPassword reports whether a password meets the length and character-class
// requirements.
func ValidPassword(s string) bool {
	if len(s) < minPasswordLen || len(s) > maxPasswordLen {
		return false
	}
	for _, re := range passwordChecks {
		if !re.MatchString(s) {
			return false
		}
	}
	return true
}

const codeAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randomCode returns n characters drawn uniformly from codeAlphabet using the
// system CSPRNG. Activation codes are password-reset tokens.
func randomCode(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// len(codeAlphabet) is 62; 256 is not a multiple of it, so reject the tail
	// of the byte range to keep the distribution uniform.
	const limit = 256 - (256 % len(codeAlphabet))
	out := make([]byte, 0, n)
	for len(out) < n {
		for _, b := range buf {
			if int(b) >= limit {
				continue
			}
			out = append(out, codeAlphabet[int(b)%len(codeAlphabet)])
			if len(out) == n {
				break
			}
		}
		if len(out) < n {
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
		}
	}
	return string(out), nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
