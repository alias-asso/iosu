package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/store/sqlc"
	"github.com/alias-asso/iosu/internal/store/storetest"
	"golang.org/x/crypto/bcrypt"
)

const testPassword = "Passw0rd!"

type testServer struct {
	*Server
	app *app.App
	t   *testing.T
}

// newTestServer builds a server over a scratch database holding one open
// contest ("alpha") with a single two-part problem ("one").
func newTestServer(t *testing.T) *testServer {
	t.Helper()
	dir := t.TempDir()
	a := app.New(storetest.New(t), dir)
	ctx := t.Context()

	if err := a.CreateDifficulty(ctx, "facile", 10); err != nil {
		t.Fatalf("difficulty: %v", err)
	}
	if _, err := a.CreateContest(ctx, app.CreateContestInput{
		Slug: "alpha", Name: "Alpha",
		StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("contest: %v", err)
	}
	if _, err := a.CreateProblem(ctx, app.CreateProblemInput{
		ContestSlug: "alpha", DifficultyName: "facile", Slug: "one",
		Name: "One", Author: "someone", Parts: 2, PointsMultiplier: 1,
	}); err != nil {
		t.Fatalf("problem: %v", err)
	}
	for _, part := range []string{"1", "2"} {
		path := filepath.Join(dir, "alpha", "one", "part"+part+".md")
		if err := os.WriteFile(path, []byte("# part "+part), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	if _, err := a.EnsureSiteConfig(ctx); err != nil {
		t.Fatalf("site config: %v", err)
	}

	srv, err := New(a, &config.Config{
		JWTKey:  strings.Repeat("k", 32),
		DataDir: dir,
		DevMode: true,
	})
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	return &testServer{Server: srv, app: a, t: t}
}

// user creates an activated account with a personal input and answers.
func (ts *testServer) user(username string) app.User {
	ts.t.Helper()
	ctx := ts.t.Context()

	if _, err := ts.app.Register(ctx, username, username+"@example.com"); err != nil {
		ts.t.Fatalf("register %s: %v", username, err)
	}
	u, err := ts.app.UserByUsername(ctx, username)
	if err != nil {
		ts.t.Fatalf("load %s: %v", username, err)
	}
	// Activating through the app would hash at the production cost, which
	// dominates the runtime of this package under -race.
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		ts.t.Fatalf("hashing: %v", err)
	}
	if err := ts.app.Store().ActivateUser(ctx, sqlc.ActivateUserParams{
		PasswordHash: string(hash), ID: u.ID,
	}); err != nil {
		ts.t.Fatalf("activate %s: %v", username, err)
	}
	u.Activated = true
	if err := ts.app.SetProblemData(ctx, u.ID, "one",
		username+"-input", []string{username + "-a1", username + "-a2"}); err != nil {
		ts.t.Fatalf("seed data for %s: %v", username, err)
	}
	return u
}

// do runs a request through the full handler chain.
func (ts *testServer) do(req *http.Request, as *app.User) *httptest.ResponseRecorder {
	ts.t.Helper()
	if as != nil {
		token, err := ts.issueToken(*as)
		if err != nil {
			ts.t.Fatalf("issuing token: %v", err)
		}
		req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	ts.Handler().ServeHTTP(rec, req)
	return rec
}

func (ts *testServer) get(path string, as *app.User) *httptest.ResponseRecorder {
	return ts.do(httptest.NewRequest(http.MethodGet, path, nil), as)
}

func (ts *testServer) postForm(path string, form url.Values, as *app.User) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return ts.do(req, as)
}

func TestProtectedRoutesRedirectAnonymousVisitors(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{
		"/contest/alpha/",
		"/contest/alpha/leaderboard",
		"/contest/alpha/one/",
		"/contest/alpha/one/input/",
		"/contest/alpha/one/img/x.png",
	} {
		t.Run(path, func(t *testing.T) {
			rec := ts.get(path, nil)
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status %d, want %d", rec.Code, http.StatusSeeOther)
			}
			if loc := rec.Header().Get("Location"); loc != "/login" {
				t.Fatalf("redirected to %q, want /login", loc)
			}
		})
	}
}

func TestPublicRoutesAreReachableAnonymously(t *testing.T) {
	ts := newTestServer(t)
	for _, path := range []string{"/", "/login", "/help", "/rules", "/legal", "/credits"} {
		t.Run(path, func(t *testing.T) {
			if rec := ts.get(path, nil); rec.Code != http.StatusOK {
				t.Fatalf("status %d, want 200", rec.Code)
			}
		})
	}
}

func TestLoginSetsAHardenedCookie(t *testing.T) {
	ts := newTestServer(t)
	ts.user("alice")

	rec := ts.postForm("/login", url.Values{
		"username": {"alice"}, "password": {testPassword},
	}, nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, want %d", rec.Code, http.StatusSeeOther)
	}

	var token *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			token = c
		}
	}
	if token == nil {
		t.Fatal("no session cookie was set")
	}
	if !token.HttpOnly {
		t.Error("cookie is not HttpOnly")
	}
	if token.SameSite != http.SameSiteStrictMode {
		t.Error("cookie is not SameSite=Strict")
	}
	// This used to be int(time.Hour*24), i.e. nanoseconds in a seconds field.
	if token.MaxAge != int(tokenTTL.Seconds()) {
		t.Errorf("MaxAge is %d, want %d seconds", token.MaxAge, int(tokenTTL.Seconds()))
	}
}

func TestLoginFailureDoesNotDistinguishUnknownFromWrong(t *testing.T) {
	ts := newTestServer(t)
	ts.user("alice")

	wrong := ts.postForm("/login", url.Values{"username": {"alice"}, "password": {"Nope!1234"}}, nil)
	unknown := ts.postForm("/login", url.Values{"username": {"nobody"}, "password": {"Nope!1234"}}, nil)

	if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
		t.Fatalf("statuses %d and %d, want 401 for both", wrong.Code, unknown.Code)
	}
	if wrong.Body.String() != unknown.Body.String() {
		t.Fatal("the two failures render differently, which lets an attacker enumerate accounts")
	}
	for _, c := range wrong.Result().Cookies() {
		if c.Name == cookieName && c.Value != "" {
			t.Fatal("a session cookie was issued for a failed login")
		}
	}
}

func TestInputIsPerUser(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")
	bob := ts.user("bob")

	for _, u := range []app.User{alice, bob} {
		rec := ts.get("/contest/alpha/one/input/", &u)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, want 200", u.Username, rec.Code)
		}
		if got := rec.Body.String(); got != u.Username+"-input" {
			t.Fatalf("%s got input %q, want their own", u.Username, got)
		}
	}
}

func TestSubmitAcceptsOnlyTheCallersOwnAnswer(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")
	ts.user("bob")

	rec := ts.postForm("/contest/alpha/one/submit/1", url.Values{"response": {"bob-a1"}}, &alice)
	if body := rec.Body.String(); !strings.Contains(body, "Mauvaise réponse") {
		t.Fatalf("another user's answer was accepted: %s", body)
	}

	rec = ts.postForm("/contest/alpha/one/submit/1", url.Values{"response": {"alice-a1"}}, &alice)
	if body := rec.Body.String(); !strings.Contains(body, "Bonne réponse") {
		t.Fatalf("the correct answer was rejected: %s", body)
	}
}

func TestSubmitRevealsTheNextPartLink(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")

	rec := ts.postForm("/contest/alpha/one/submit/1", url.Values{"response": {"alice-a1"}}, &alice)
	if !strings.Contains(rec.Body.String(), "partie suivante") {
		t.Fatal("solving part 1 of 2 should offer the next part")
	}
	rec = ts.postForm("/contest/alpha/one/submit/2", url.Values{"response": {"alice-a2"}}, &alice)
	if strings.Contains(rec.Body.String(), "partie suivante") {
		t.Fatal("solving the last part should not offer another one")
	}
}

func TestImageTraversalIsRejected(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")

	for _, path := range []string{
		"/contest/alpha/one/img/" + url.PathEscape("../../../etc/passwd"),
		"/contest/%2e%2e%2f%2e%2e%2fetc/one/img/passwd",
		"/contest/alpha/%2e%2e/img/passwd",
	} {
		t.Run(path, func(t *testing.T) {
			rec := ts.get(path, &alice)
			if rec.Code == http.StatusOK {
				t.Fatalf("status 200, want a rejection; body: %q", rec.Body.String())
			}
		})
	}
}

func TestTamperedTokenIsRejected(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")

	token, err := ts.issueToken(alice)
	if err != nil {
		t.Fatalf("issuing token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/contest/alpha/one/input/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: token + "x"})

	rec := httptest.NewRecorder()
	ts.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, want a redirect to the login page", rec.Code)
	}
}

func TestTokenSignedWithAnotherKeyIsRejected(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")

	other := &Server{cfg: &config.Config{JWTKey: strings.Repeat("z", 32)}}
	forged, err := other.issueToken(alice)
	if err != nil {
		t.Fatalf("forging: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/contest/alpha/one/input/", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: forged})

	rec := httptest.NewRecorder()
	ts.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, want a redirect: a token signed with another key must not authenticate", rec.Code)
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	ts := newTestServer(t)
	h := ts.get("/", nil).Header()

	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	} {
		if got := h.Get(header); got != want {
			t.Errorf("%s is %q, want %q", header, got, want)
		}
	}
	if h.Get("Content-Security-Policy") == "" {
		t.Error("no Content-Security-Policy")
	}
}

func TestPanicsAreRecovered(t *testing.T) {
	rec := httptest.NewRecorder()
	handler := recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func TestErrorsAreShownInFrenchWithoutInternals(t *testing.T) {
	ts := newTestServer(t)
	alice := ts.user("alice")

	rec := ts.get("/contest/alpha/nosuchproblem/", &alice)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Problème introuvable") {
		t.Fatalf("body does not carry the French message: %s", rec.Body.String())
	}
}

// setCurrentContest points the site config at slug, or clears it when empty.
func (ts *testServer) setCurrentContest(slug string) {
	ts.t.Helper()
	if err := ts.app.UpdateSiteConfig(ts.t.Context(), sqlc.UpdateSiteConfigParams{
		CurrentContest: sql.NullString{String: slug, Valid: true},
	}); err != nil {
		ts.t.Fatalf("setting current contest %q: %v", slug, err)
	}
}

func TestHomePageFallsBackToTheArchive(t *testing.T) {
	ts := newTestServer(t)
	ts.setCurrentContest("")

	body := ts.get("/", nil).Body.String()
	for _, want := range []string{"Rejouer les anciens concours", "Alpha", "/contest/alpha/"} {
		if !strings.Contains(body, want) {
			t.Errorf("home page does not mention %q", want)
		}
	}
	for _, unwanted := range []string{"Accéder au concours", "/contest//"} {
		if strings.Contains(body, unwanted) {
			t.Errorf("home page still carries %q with no active contest", unwanted)
		}
	}
}

func TestHomePageShowsTheActiveContest(t *testing.T) {
	ts := newTestServer(t)
	ts.setCurrentContest("alpha")

	body := ts.get("/", nil).Body.String()
	for _, want := range []string{"<h2>Alpha</h2>", "Accéder au concours", "/contest/alpha/leaderboard"} {
		if !strings.Contains(body, want) {
			t.Errorf("home page does not carry %q", want)
		}
	}
	if strings.Contains(body, "Rejouer les anciens concours") {
		t.Error("home page lists the archive while a contest is active")
	}
}

func TestArchiveIsPublic(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.get("/archive", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Alpha") {
		t.Errorf("archive does not list the contest: %s", rec.Body.String())
	}
}

func TestTemplatesAllParse(t *testing.T) {
	// New builds every template up front, so this fails at startup rather than
	// mid-request if a page is malformed.
	if _, err := parseTemplates(); err != nil {
		t.Fatalf("parsing templates: %v", err)
	}
}
