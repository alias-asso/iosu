package app

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/store/sqlc"
)

// window creates a contest running between the two offsets from the fixture
// clock.
func (f *fixture) window(slug string, start, end time.Duration) Contest {
	f.t.Helper()
	c, err := f.CreateContest(f.ctx(), CreateContestInput{
		Slug:      slug,
		Name:      slug,
		StartTime: f.clock.Add(start),
		EndTime:   f.clock.Add(end),
	})
	if err != nil {
		f.t.Fatalf("creating contest %s: %v", slug, err)
	}
	return c
}

func (f *fixture) unlist(c Contest) {
	f.t.Helper()
	if err := f.UpdateContest(f.ctx(), sqlc.UpdateContestParams{
		ID:       c.ID,
		Unlisted: sql.NullBool{Bool: true, Valid: true},
	}); err != nil {
		f.t.Fatalf("unlisting %s: %v", c.Slug, err)
	}
}

func TestArchiveListsVisibleContestsNewestFirst(t *testing.T) {
	f := newFixture(t)
	day := 24 * time.Hour
	f.window("past", -3*day, -2*day)
	f.window("running", -time.Hour, time.Hour)
	f.window("upcoming", day, 2*day)
	f.unlist(f.window("hidden", 3*day, 4*day))

	entries, err := f.Archive(f.ctx())
	if err != nil {
		t.Fatalf("archive: %v", err)
	}

	want := []struct {
		slug   string
		status ContestStatus
	}{
		{"upcoming", ContestUpcoming},
		{"running", ContestRunning},
		{"past", ContestFinished},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(entries), len(want), entries)
	}
	for i, w := range want {
		if entries[i].Contest.Slug != w.slug {
			t.Errorf("entry %d: got %q, want %q", i, entries[i].Contest.Slug, w.slug)
		}
		if entries[i].Status != w.status {
			t.Errorf("%s: got status %q, want %q", w.slug, entries[i].Status, w.status)
		}
	}
}

func TestUpdateSiteConfigChecksTheCurrentContest(t *testing.T) {
	f := newFixture(t)
	if _, err := f.EnsureSiteConfig(f.ctx()); err != nil {
		t.Fatalf("site config: %v", err)
	}
	f.contest("alpha")

	set := func(slug string) error {
		return f.UpdateSiteConfig(f.ctx(), sqlc.UpdateSiteConfigParams{
			CurrentContest: sql.NullString{String: slug, Valid: true},
		})
	}

	if err := set("alpha"); err != nil {
		t.Fatalf("setting a real contest: %v", err)
	}

	if err := set("nope"); !errors.Is(err, ErrContestNotFound) {
		t.Errorf("setting an unknown contest: got %v, want ErrContestNotFound", err)
	}
	config, err := f.SiteConfig(f.ctx())
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if config.CurrentContest != "alpha" {
		t.Errorf("rejected update still changed the config: %q", config.CurrentContest)
	}

	// The empty string is how the active contest is turned off.
	if err := set(""); err != nil {
		t.Fatalf("clearing the current contest: %v", err)
	}
	if config, err = f.SiteConfig(f.ctx()); err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if config.CurrentContest != "" {
		t.Errorf("current contest not cleared: %q", config.CurrentContest)
	}
}

func TestUpdateContestRejectsReservedSlug(t *testing.T) {
	f := newFixture(t)
	contest := f.contest("alpha")
	err := f.UpdateContest(f.ctx(), sqlc.UpdateContestParams{
		ID:   contest.ID,
		Slug: sql.NullString{String: "new", Valid: true},
	})
	if !errors.Is(err, ErrInvalidSlug) {
		t.Fatalf("got %v, want ErrInvalidSlug", err)
	}
}

func TestDeleteContest(t *testing.T) {
	f := newFixture(t)
	contest := f.contest("alpha")
	if _, err := f.EnsureSiteConfig(f.ctx()); err != nil {
		t.Fatalf("site config: %v", err)
	}
	if err := f.UpdateSiteConfig(f.ctx(), sqlc.UpdateSiteConfigParams{
		CurrentContest: sql.NullString{String: contest.Slug, Valid: true},
	}); err != nil {
		t.Fatalf("setting current contest: %v", err)
	}

	if err := f.DeleteContest(f.ctx(), contest.Slug); err != nil {
		t.Fatalf("deleting contest: %v", err)
	}
	if _, err := f.Contest(f.ctx(), contest.Slug); !errors.Is(err, ErrContestNotFound) {
		t.Fatalf("loading deleted contest: got %v, want ErrContestNotFound", err)
	}
	config, err := f.SiteConfig(f.ctx())
	if err != nil {
		t.Fatalf("site config: %v", err)
	}
	if config.CurrentContest != "" {
		t.Errorf("current contest is %q after deletion, want empty", config.CurrentContest)
	}
	if err := f.DeleteContest(f.ctx(), contest.Slug); !errors.Is(err, ErrContestNotFound) {
		t.Fatalf("deleting missing contest: got %v, want ErrContestNotFound", err)
	}
}

func TestDeleteContestKeepsItsProblems(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", 1)

	if err := f.DeleteContest(f.ctx(), "alpha"); !errors.Is(err, ErrContestNotEmpty) {
		t.Fatalf("deleting non-empty contest: got %v, want ErrContestNotEmpty", err)
	}
	if _, err := f.Contest(f.ctx(), "alpha"); err != nil {
		t.Fatalf("contest was deleted: %v", err)
	}
	if _, err := f.Problem(f.ctx(), "one"); err != nil {
		t.Fatalf("problem was deleted: %v", err)
	}
}
