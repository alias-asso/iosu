package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/store/storetest"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	// Real cost makes the fixtures in this package take tens of seconds.
	bcryptCost = bcrypt.MinCost
	os.Exit(m.Run())
}

// fixture is a fresh app with a data directory, plus a clock the test controls.
type fixture struct {
	*App
	clock time.Time
	dir   string
	t     *testing.T
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	a := New(storetest.New(t), dir)
	f := &fixture{App: a, clock: time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC), dir: dir, t: t}
	a.now = func() time.Time { return f.clock }
	return f
}

func (f *fixture) ctx() context.Context { return f.t.Context() }

// contest creates a contest that is open at the fixture's current time.
func (f *fixture) contest(slug string) Contest {
	f.t.Helper()
	c, err := f.CreateContest(f.ctx(), CreateContestInput{
		Slug:      slug,
		Name:      slug,
		StartTime: f.clock.Add(-time.Hour),
		EndTime:   f.clock.Add(time.Hour),
	})
	if err != nil {
		f.t.Fatalf("creating contest %s: %v", slug, err)
	}
	return c
}

func (f *fixture) difficulty(name string, points int64) {
	f.t.Helper()
	if err := f.CreateDifficulty(f.ctx(), name, points); err != nil {
		f.t.Fatalf("creating difficulty %s: %v", name, err)
	}
}

// problem creates a problem with parts markdown files on disk.
func (f *fixture) problem(contestSlug, slug string, parts int64) Problem {
	f.t.Helper()
	p, err := f.CreateProblem(f.ctx(), CreateProblemInput{
		ContestSlug:      contestSlug,
		DifficultyName:   "facile",
		Slug:             slug,
		Name:             slug,
		Author:           "someone",
		Parts:            parts,
		PointsMultiplier: 1,
	})
	if err != nil {
		f.t.Fatalf("creating problem %s: %v", slug, err)
	}
	for i := int64(1); i <= parts; i++ {
		path := filepath.Join(f.dir, contestSlug, slug, "part"+itoa(i)+".md")
		if err := os.WriteFile(path, []byte("# part "+itoa(i)), 0o644); err != nil {
			f.t.Fatalf("writing %s: %v", path, err)
		}
	}
	return p
}

// user creates an activated account and returns it.
func (f *fixture) user(username string) User {
	f.t.Helper()
	code, err := f.Register(f.ctx(), username, username+"@example.com")
	if err != nil {
		f.t.Fatalf("registering %s: %v", username, err)
	}
	if err := f.Activate(f.ctx(), code, "Passw0rd!"); err != nil {
		f.t.Fatalf("activating %s: %v", username, err)
	}
	u, err := f.UserByUsername(f.ctx(), username)
	if err != nil {
		f.t.Fatalf("loading %s: %v", username, err)
	}
	return u
}

func itoa(n int64) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + itoa(n%10)
}
