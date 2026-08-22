package app

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAuthenticate(t *testing.T) {
	f := newFixture(t)
	f.user("alice")

	tests := []struct {
		name     string
		username string
		password string
		want     error
	}{
		{"correct", "alice", "Passw0rd!", nil},
		{"wrong password", "alice", "Wr0ngPass!", ErrInvalidCredentials},
		{"unknown user", "nobody", "Passw0rd!", ErrInvalidCredentials},
		{"empty username", "", "Passw0rd!", ErrInvalidCredentials},
		{"empty password", "alice", "", ErrInvalidCredentials},
		{"username is case-insensitive", "ALICE", "Passw0rd!", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.Authenticate(f.ctx(), tc.username, tc.password)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestAuthenticateRejectsUnactivatedAccount(t *testing.T) {
	f := newFixture(t)
	if _, err := f.Register(f.ctx(), "bob", "bob@example.com"); err != nil {
		t.Fatalf("register: %v", err)
	}
	// An unactivated account has an empty password hash; the empty password
	// must not be accepted either.
	if _, err := f.Authenticate(f.ctx(), "bob", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("empty password: got %v, want ErrInvalidCredentials", err)
	}
	if _, err := f.Authenticate(f.ctx(), "bob", "Passw0rd!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestRegisterRejectsDuplicates(t *testing.T) {
	f := newFixture(t)
	if _, err := f.Register(f.ctx(), "alice", "alice@example.com"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if _, err := f.Register(f.ctx(), "alice", "other@example.com"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate username: got %v, want ErrUserExists", err)
	}
	if _, err := f.Register(f.ctx(), "other", "alice@example.com"); !errors.Is(err, ErrUserExists) {
		t.Fatalf("duplicate email: got %v, want ErrUserExists", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name     string
		username string
		email    string
		want     error
	}{
		{"empty username", "", "a@example.com", ErrInvalidUsername},
		{"long username", strings.Repeat("a", maxUsernameLen+1), "a@example.com", ErrInvalidUsername},
		{"bad email", "alice", "not-an-email", ErrInvalidEmail},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.Register(f.ctx(), tc.username, tc.email); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestActivate(t *testing.T) {
	f := newFixture(t)
	code, err := f.Register(f.ctx(), "alice", "alice@example.com")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := f.Activate(f.ctx(), code, "weak"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("weak password: got %v, want ErrWeakPassword", err)
	}
	if err := f.Activate(f.ctx(), code, "Passw0rd!"); err != nil {
		t.Fatalf("activate: %v", err)
	}
	// A link is a password-reset token: it must work exactly once.
	if err := f.Activate(f.ctx(), code, "An0therPass!"); !errors.Is(err, ErrActivationCodeUsed) {
		t.Fatalf("reuse: got %v, want ErrActivationCodeUsed", err)
	}
	// The first password must still be the live one.
	if _, err := f.Authenticate(f.ctx(), "alice", "Passw0rd!"); err != nil {
		t.Fatalf("login after replayed activation: %v", err)
	}
}

func TestActivateExpiredCode(t *testing.T) {
	f := newFixture(t)
	code, err := f.Register(f.ctx(), "alice", "alice@example.com")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	f.clock = f.clock.Add(activationTTL + time.Minute)

	if err := f.Activate(f.ctx(), code, "Passw0rd!"); !errors.Is(err, ErrActivationCodeExpired) {
		t.Fatalf("got %v, want ErrActivationCodeExpired", err)
	}
}

func TestActivateUnknownCode(t *testing.T) {
	f := newFixture(t)
	for _, code := range []string{"", "short", strings.Repeat("x", activationCodeLen)} {
		if err := f.Activate(f.ctx(), code, "Passw0rd!"); !errors.Is(err, ErrInvalidActivationCode) {
			t.Fatalf("code %q: got %v, want ErrInvalidActivationCode", code, err)
		}
	}
}

func TestBatchRegister(t *testing.T) {
	f := newFixture(t)

	n, err := f.BatchRegister(f.ctx(), "username,email\nalice,alice@example.com\nbob,bob@example.com\n")
	if err != nil {
		t.Fatalf("batch register: %v", err)
	}
	if n != 2 {
		t.Fatalf("created %d accounts, want 2", n)
	}
	pending, err := f.PendingActivations(f.ctx())
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("got %d pending activations, want 2", len(pending))
	}
}

func TestBatchRegisterIsAllOrNothing(t *testing.T) {
	f := newFixture(t)
	f.user("alice")

	// The second row collides with the existing account.
	_, err := f.BatchRegister(f.ctx(), "username,email\nzoe,zoe@example.com\nalice,dup@example.com\n")
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("got %v, want ErrUserExists", err)
	}
	if _, err := f.UserByUsername(f.ctx(), "zoe"); !errors.Is(err, ErrUserNotFound) {
		t.Fatal("the first row was committed even though the batch failed")
	}
}

func TestBatchRegisterRejectsBadCSV(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name string
		body string
		want error
	}{
		{"empty", "", ErrInvalidCSV},
		{"missing columns", "name,mail\nalice,a@example.com\n", ErrInvalidCSV},
		{"bad email", "username,email\nalice,nope\n", ErrInvalidEmail},
		{"empty username", "username,email\n,a@example.com\n", ErrInvalidUsername},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.BatchRegister(f.ctx(), tc.body); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestEnsureAdminOnlyOnAnEmptyDatabase(t *testing.T) {
	f := newFixture(t)

	created, err := f.EnsureAdmin(f.ctx(), "Adm1nPass!")
	if err != nil || !created {
		t.Fatalf("first call: created=%v err=%v", created, err)
	}
	if err := f.SetPassword(f.ctx(), "admin", "Ch4ngedPass!"); err != nil {
		t.Fatalf("set password: %v", err)
	}

	// A restart must not put the configured password back.
	created, err = f.EnsureAdmin(f.ctx(), "Adm1nPass!")
	if err != nil || created {
		t.Fatalf("second call: created=%v err=%v", created, err)
	}
	if _, err := f.Authenticate(f.ctx(), "admin", "Ch4ngedPass!"); err != nil {
		t.Fatalf("changed password no longer works: %v", err)
	}
	if _, err := f.Authenticate(f.ctx(), "admin", "Adm1nPass!"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("the config password was restored over the changed one")
	}
}

func TestSetAdmin(t *testing.T) {
	f := newFixture(t)
	u := f.user("alice")
	if u.Admin {
		t.Fatal("a new account must not be an admin")
	}

	if err := f.SetAdmin(f.ctx(), "alice", true); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if u, _ := f.UserByUsername(f.ctx(), "alice"); !u.Admin {
		t.Fatal("admin was not granted")
	}
	if err := f.SetAdmin(f.ctx(), "alice", false); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if u, _ := f.UserByUsername(f.ctx(), "alice"); u.Admin {
		t.Fatal("admin was not revoked")
	}
}

func TestValidPassword(t *testing.T) {
	tests := []struct {
		password string
		want     bool
	}{
		{"Passw0rd!", true},
		{"Sh0rt!", false},                   // under 8 characters
		{"passw0rd!", false},                // no uppercase
		{"PASSW0RD!", false},                // no lowercase
		{"Password!", false},                // no digit
		{"Passw0rdd", false},                // no special character
		{strings.Repeat("Aa1!", 20), false}, // over 72 bytes, where bcrypt truncates
	}
	for _, tc := range tests {
		t.Run(tc.password, func(t *testing.T) {
			if got := ValidPassword(tc.password); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRandomCode(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for range 1000 {
		code, err := randomCode(activationCodeLen)
		if err != nil {
			t.Fatalf("randomCode: %v", err)
		}
		if len(code) != activationCodeLen {
			t.Fatalf("length %d, want %d", len(code), activationCodeLen)
		}
		if seen[code] {
			t.Fatalf("duplicate code %q", code)
		}
		seen[code] = true
		for _, r := range code {
			if !strings.ContainsRune(codeAlphabet, r) {
				t.Fatalf("code %q contains %q, which is outside the alphabet", code, r)
			}
		}
	}
}

func TestLeaderboardIsScopedToItsContest(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.contest("beta")
	f.problem("alpha", "alpha-one", 1)
	f.problem("beta", "beta-one", 1)
	alice := f.user("alice")

	for _, slug := range []string{"alpha-one", "beta-one"} {
		if err := f.SetProblemData(f.ctx(), alice.ID, slug, "in", []string{"42"}); err != nil {
			t.Fatalf("seeding %s: %v", slug, err)
		}
		ok, err := f.Submit(f.ctx(), SubmitInput{
			UserID: alice.ID, ContestSlug: strings.Split(slug, "-")[0],
			ProblemSlug: slug, Part: 1, Answer: "42",
		})
		if err != nil || !ok {
			t.Fatalf("solving %s: ok=%v err=%v", slug, ok, err)
		}
	}

	board, err := f.Leaderboard(f.ctx(), "alpha")
	if err != nil {
		t.Fatalf("leaderboard: %v", err)
	}
	if len(board) != 1 {
		t.Fatalf("got %d rows, want 1", len(board))
	}
	// 10 points for one part of one problem, not both contests' 20.
	if board[0].Score != 10 {
		t.Fatalf("score %v, want 10 (the other contest leaked in)", board[0].Score)
	}
}

func TestLeaderboardExcludesAdminsAndZeroScores(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", 1)
	f.user("zero")
	admin := f.user("root")

	if err := f.SetAdmin(f.ctx(), "root", true); err != nil {
		t.Fatalf("set admin: %v", err)
	}
	if err := f.SetProblemData(f.ctx(), admin.ID, "one", "in", []string{"42"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := f.Submit(f.ctx(), SubmitInput{
		UserID: admin.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "42",
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}

	board, err := f.Leaderboard(f.ctx(), "alpha")
	if err != nil {
		t.Fatalf("leaderboard: %v", err)
	}
	if len(board) != 0 {
		t.Fatalf("got %d rows, want 0: admins and users without points must be left out", len(board))
	}
}
