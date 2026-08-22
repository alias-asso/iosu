package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// seedProblem sets up a one-contest, one-problem world with data for alice.
func seedProblem(t *testing.T, parts int64, outputs []string) (*fixture, User) {
	t.Helper()
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", parts)
	alice := f.user("alice")
	if err := f.SetProblemData(f.ctx(), alice.ID, "one", "the input", outputs); err != nil {
		t.Fatalf("seeding problem data: %v", err)
	}
	return f, alice
}

func TestSubmitCorrectAnswer(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})

	ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "42",
	})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}

	p, _ := f.Problem(f.ctx(), "one")
	solved, err := f.SolvedParts(f.ctx(), alice.ID, p.Problem.ID)
	if err != nil {
		t.Fatalf("solved parts: %v", err)
	}
	if solved != 1 {
		t.Fatalf("solved %d parts, want 1", solved)
	}
}

func TestSubmitWrongAnswerIsNotAnError(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})

	ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "43",
	})
	if err != nil {
		t.Fatalf("a wrong answer must not be an error: %v", err)
	}
	if ok {
		t.Fatal("a wrong answer was accepted")
	}
}

func TestSubmitTrimsWhitespace(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})

	// A trailing newline from the contestant's clipboard used to be a silent
	// wrong answer, because only the import side trimmed.
	ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "  42\n",
	})
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestSubmitPartOrdering(t *testing.T) {
	f, alice := seedProblem(t, 2, []string{"one", "two"})
	submit := func(part int64, answer string) (bool, error) {
		return f.Submit(f.ctx(), SubmitInput{
			UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: part, Answer: answer,
		})
	}

	if _, err := submit(2, "two"); !errors.Is(err, ErrInvalidPart) {
		t.Fatalf("skipping to part 2: got %v, want ErrInvalidPart", err)
	}
	if ok, err := submit(1, "one"); err != nil || !ok {
		t.Fatalf("part 1: ok=%v err=%v", ok, err)
	}
	if _, err := submit(1, "one"); !errors.Is(err, ErrAlreadySolved) {
		t.Fatalf("resubmitting part 1: got %v, want ErrAlreadySolved", err)
	}
	if ok, err := submit(2, "two"); err != nil || !ok {
		t.Fatalf("part 2: ok=%v err=%v", ok, err)
	}
}

func TestSubmitRejectsOutOfRangeParts(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	for _, part := range []int64{0, -1, 2, 99} {
		_, err := f.Submit(f.ctx(), SubmitInput{
			UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: part, Answer: "42",
		})
		if !errors.Is(err, ErrInvalidPart) {
			t.Fatalf("part %d: got %v, want ErrInvalidPart", part, err)
		}
	}
}

func TestSubmitChecksContestWindow(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	submit := func() error {
		_, err := f.Submit(f.ctx(), SubmitInput{
			UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "42",
		})
		return err
	}

	f.clock = f.clock.Add(-2 * time.Hour)
	if err := submit(); !errors.Is(err, ErrContestNotStarted) {
		t.Fatalf("before start: got %v, want ErrContestNotStarted", err)
	}
	f.clock = f.clock.Add(4 * time.Hour)
	if err := submit(); !errors.Is(err, ErrContestFinished) {
		t.Fatalf("after end: got %v, want ErrContestFinished", err)
	}
}

func TestSubmitRejectsContestMismatch(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	f.contest("beta")

	// The URL claims the problem belongs to beta; it does not.
	_, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "beta", ProblemSlug: "one", Part: 1, Answer: "42",
	})
	if !errors.Is(err, ErrProblemNotFound) {
		t.Fatalf("got %v, want ErrProblemNotFound", err)
	}
}

func TestSubmitWithoutSeededOutput(t *testing.T) {
	f, _ := seedProblem(t, 1, []string{"42"})
	bob := f.user("bob")

	_, err := f.Submit(f.ctx(), SubmitInput{
		UserID: bob.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "42",
	})
	if !errors.Is(err, ErrOutputNotFound) {
		t.Fatalf("got %v, want ErrOutputNotFound", err)
	}
}

func TestSubmitUsesEachUsersOwnAnswer(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"alice-answer"})
	bob := f.user("bob")
	if err := f.SetProblemData(f.ctx(), bob.ID, "one", "bob input", []string{"bob-answer"}); err != nil {
		t.Fatalf("seeding bob: %v", err)
	}

	// Alice's answer must not unlock the problem for Bob.
	ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: bob.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "alice-answer",
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if ok {
		t.Fatal("one contestant's answer was accepted for another")
	}
	if ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "alice-answer",
	}); err != nil || !ok {
		t.Fatalf("alice's own answer: ok=%v err=%v", ok, err)
	}
}

func TestSubmitConcurrentlyRecordsOneSolve(t *testing.T) {
	f, alice := seedProblem(t, 2, []string{"one", "two"})

	// The old code read the solve count, then wrote, with no constraint in
	// between: two racing submissions each inserted a row.
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.Submit(f.ctx(), SubmitInput{
				UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "one",
			})
		}()
	}
	wg.Wait()

	p, _ := f.Problem(f.ctx(), "one")
	solved, err := f.SolvedParts(f.ctx(), alice.ID, p.Problem.ID)
	if err != nil {
		t.Fatalf("solved parts: %v", err)
	}
	if solved != 1 {
		t.Fatalf("solved %d parts, want exactly 1", solved)
	}
}

func TestProblemStatementRevealsPartsAsTheyAreSolved(t *testing.T) {
	f, alice := seedProblem(t, 3, []string{"one", "two", "three"})
	detail, _ := f.Problem(f.ctx(), "one")

	parts, err := f.ProblemStatement(f.ctx(), alice.ID, detail)
	if err != nil {
		t.Fatalf("statement: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("got %d parts before solving anything, want 1", len(parts))
	}

	if _, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "one",
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	parts, err = f.ProblemStatement(f.ctx(), alice.ID, detail)
	if err != nil {
		t.Fatalf("statement: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("got %d parts after solving part 1, want 2", len(parts))
	}
}

func TestProblemStatementChecksContestWindow(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	detail, _ := f.Problem(f.ctx(), "one")

	f.clock = f.clock.Add(-2 * time.Hour)
	if _, err := f.ProblemStatement(f.ctx(), alice.ID, detail); !errors.Is(err, ErrContestNotStarted) {
		t.Fatalf("got %v, want ErrContestNotStarted", err)
	}
}

func TestProblemInputIsGatedOnTheContestWindow(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	detail, _ := f.Problem(f.ctx(), "one")

	if got, err := f.ProblemInput(f.ctx(), alice.ID, detail); err != nil || got != "the input" {
		t.Fatalf("during the contest: got %q err=%v", got, err)
	}

	// Downloading the input was the one path that never checked the window.
	f.clock = f.clock.Add(-2 * time.Hour)
	if _, err := f.ProblemInput(f.ctx(), alice.ID, detail); !errors.Is(err, ErrContestNotStarted) {
		t.Fatalf("before the contest: got %v, want ErrContestNotStarted", err)
	}
}

func TestProblemInputIsPerUser(t *testing.T) {
	f, _ := seedProblem(t, 1, []string{"42"})
	bob := f.user("bob")
	detail, _ := f.Problem(f.ctx(), "one")

	if _, err := f.ProblemInput(f.ctx(), bob.ID, detail); !errors.Is(err, ErrInputNotFound) {
		t.Fatalf("got %v, want ErrInputNotFound", err)
	}
}

func TestSetProblemDataChecksPartCount(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", 2)
	alice := f.user("alice")

	if err := f.SetProblemData(f.ctx(), alice.ID, "one", "in", []string{"only-one"}); !errors.Is(err, ErrPartCountMismatch) {
		t.Fatalf("got %v, want ErrPartCountMismatch", err)
	}
}

func TestSetProblemDataOverwrites(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	if err := f.SetProblemData(f.ctx(), alice.ID, "one", "new input", []string{"99"}); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	detail, _ := f.Problem(f.ctx(), "one")
	if got, _ := f.ProblemInput(f.ctx(), alice.ID, detail); got != "new input" {
		t.Fatalf("input is %q, want the re-imported value", got)
	}
	if ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "99",
	}); err != nil || !ok {
		t.Fatalf("new answer: ok=%v err=%v", ok, err)
	}
}

func TestCreateProblemValidation(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")

	base := CreateProblemInput{
		ContestSlug: "alpha", DifficultyName: "facile",
		Slug: "ok", Name: "Ok", Parts: 1, PointsMultiplier: 1,
	}
	tests := []struct {
		name   string
		mutate func(*CreateProblemInput)
		want   error
	}{
		{"path traversal in slug", func(in *CreateProblemInput) { in.Slug = "../../etc" }, ErrInvalidSlug},
		{"slash in slug", func(in *CreateProblemInput) { in.Slug = "a/b" }, ErrInvalidSlug},
		{"uppercase slug", func(in *CreateProblemInput) { in.Slug = "Nope" }, ErrInvalidSlug},
		{"empty slug", func(in *CreateProblemInput) { in.Slug = "" }, ErrInvalidSlug},
		{"empty name", func(in *CreateProblemInput) { in.Name = "" }, ErrInvalidName},
		{"long name", func(in *CreateProblemInput) { in.Name = strings.Repeat("a", maxNameLen+1) }, ErrInvalidName},
		{"long author", func(in *CreateProblemInput) { in.Author = strings.Repeat("a", maxAuthorLen+1) }, ErrInvalidName},
		{"zero parts", func(in *CreateProblemInput) { in.Parts = 0 }, ErrInvalidPart},
		{"unknown contest", func(in *CreateProblemInput) { in.ContestSlug = "nope" }, ErrContestNotFound},
		{"unknown difficulty", func(in *CreateProblemInput) { in.DifficultyName = "nope" }, ErrDifficultyNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			if _, err := f.CreateProblem(f.ctx(), in); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCreateProblemRejectsDuplicateSlug(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", 1)

	_, err := f.CreateProblem(f.ctx(), CreateProblemInput{
		ContestSlug: "alpha", DifficultyName: "facile", Slug: "one", Name: "Dup", Parts: 1, PointsMultiplier: 1,
	})
	if !errors.Is(err, ErrProblemExists) {
		t.Fatalf("got %v, want ErrProblemExists", err)
	}
}

func TestCreateContestValidation(t *testing.T) {
	f := newFixture(t)
	now := f.clock

	tests := []struct {
		name string
		in   CreateContestInput
		want error
	}{
		{"traversal slug", CreateContestInput{Slug: "../etc", Name: "x", StartTime: now, EndTime: now}, ErrInvalidSlug},
		{"empty name", CreateContestInput{Slug: "ok", Name: "", StartTime: now, EndTime: now}, ErrInvalidName},
		{"end before start", CreateContestInput{Slug: "ok", Name: "x", StartTime: now, EndTime: now.Add(-time.Hour)}, ErrInvalidTimeRange},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.CreateContest(f.ctx(), tc.in); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestProblemImageStaysInsideTheDataDirectory(t *testing.T) {
	f := newFixture(t)

	// A file that a traversal would reach if the guard were broken.
	secret := filepath.Join(filepath.Dir(f.dir), "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o600); err != nil {
		t.Fatalf("writing bait: %v", err)
	}

	tests := []struct {
		name                  string
		contest, problem, img string
	}{
		{"traversal in the image name", "alpha", "one", "../../../secret.txt"},
		{"slash in the image name", "alpha", "one", "sub/evil.png"},
		{"backslash in the image name", "alpha", "one", `sub\evil.png`},
		{"traversal in the contest slug", "../..", "one", "secret.txt"},
		{"traversal in the problem slug", "alpha", "../../..", "secret.txt"},
		{"absolute image name", "alpha", "one", "/etc/passwd"},
		{"empty image name", "alpha", "one", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := f.ProblemImage(tc.contest, tc.problem, tc.img); err == nil {
				t.Fatalf("resolved to %q, want a rejection", got)
			}
		})
	}
}

func TestProblemImageResolvesValidNames(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", 1)

	dir := filepath.Join(f.dir, "alpha", "one", "img")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	want := filepath.Join(dir, "figure.png")
	if err := os.WriteFile(want, []byte("png"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := f.ProblemImage("alpha", "one", "figure.png")
	if err != nil {
		t.Fatalf("resolving a valid image: %v", err)
	}
	if resolved, _ := filepath.EvalSymlinks(got); resolved != mustEval(t, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func mustEval(t *testing.T, path string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("evaluating %s: %v", path, err)
	}
	return p
}

func TestProblemsRefusesOutsideTheContestWindow(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.problem("alpha", "one", 1)

	if problems, err := f.Problems(f.ctx(), "alpha"); err != nil || len(problems) != 1 {
		t.Fatalf("during the contest: %d problems, err=%v", len(problems), err)
	}
	f.clock = f.clock.Add(2 * time.Hour)
	if _, err := f.Problems(f.ctx(), "alpha"); !errors.Is(err, ErrContestFinished) {
		t.Fatalf("after the contest: got %v, want ErrContestFinished", err)
	}
}

func TestProblemsAreScopedToTheirContest(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	f.contest("alpha")
	f.contest("beta")
	f.problem("alpha", "alpha-one", 1)
	f.problem("beta", "beta-one", 1)
	f.problem("beta", "beta-two", 1)

	// The old GetAll ignored its contest argument and returned everything.
	problems, err := f.Problems(f.ctx(), "alpha")
	if err != nil {
		t.Fatalf("problems: %v", err)
	}
	if len(problems) != 1 {
		t.Fatalf("got %d problems for alpha, want 1", len(problems))
	}
	if problems[0].Problem.Slug != "alpha-one" {
		t.Fatalf("got %q, want alpha-one", problems[0].Problem.Slug)
	}
}
