package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/store/sqlc"
)

func TestAutomaticGenerationRunsScriptsAndStoresCompleteData(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	_, err := f.CreateContest(f.ctx(), CreateContestInput{
		Slug: "alpha", Name: "alpha", StartTime: f.clock.Add(-time.Hour),
		EndTime: f.clock.Add(time.Hour), AutoGenerate: true,
	})
	if err != nil {
		t.Fatalf("creating contest: %v", err)
	}
	f.problem("alpha", "one", 2)
	f.generator("alpha", "one", "generate_input", "#!/bin/sh\nprintf '%s-data\\n' \"$1\"\n")
	f.generator("alpha", "one", "generate_output1", "#!/bin/sh\nprintf '%s-one\\n' \"$1\"\n")
	f.generator("alpha", "one", "generate_output2", "#!/bin/sh\nprintf '%s-two\\n' \"$1\"\n")

	code, err := f.Register(f.ctx(), "alice", "alice@example.com")
	if err != nil {
		t.Fatalf("registering: %v", err)
	}
	if err := f.Activate(f.ctx(), code, "Passw0rd!"); err != nil {
		t.Fatalf("activating: %v", err)
	}
	alice, _ := f.UserByUsername(f.ctx(), "alice")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.RunGenerator(ctx, testGenerationConfig()) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	detail, _ := f.Problem(f.ctx(), "one")
	waitFor(t, func() bool {
		input, err := f.ProblemInput(f.ctx(), alice.ID, detail)
		return err == nil && input == "alice-data"
	})
	for part, answer := range []string{"alice-data-one", "alice-data-two"} {
		ok, err := f.Submit(f.ctx(), SubmitInput{
			UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one",
			Part: int64(part + 1), Answer: answer,
		})
		if err != nil || !ok {
			t.Fatalf("part %d: ok=%v err=%v", part+1, ok, err)
		}
	}
	runs, err := f.GenerationRuns(f.ctx(), 10)
	if err != nil || len(runs) != 1 || runs[0].Status != "completed" || runs[0].Succeeded != 1 {
		t.Fatalf("runs=%+v err=%v", runs, err)
	}
}

func TestFailedGenerationPreservesExistingData(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"old-answer"})
	f.generator("alpha", "one", "generate_input", "#!/bin/sh\nprintf new-input\n")
	f.generator("alpha", "one", "generate_output1", "#!/bin/sh\necho broken >&2\nexit 2\n")
	runID, err := f.QueueGeneration(f.ctx(), "alpha", "replace", []int64{alice.ID})
	if err != nil {
		t.Fatalf("queueing: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.RunGenerator(ctx, testGenerationConfig()) }()
	defer func() { cancel(); <-done }()
	waitFor(t, func() bool {
		run, _, err := f.GenerationRun(f.ctx(), runID)
		return err == nil && run.Status == "completed_with_errors"
	})
	detail, _ := f.Problem(f.ctx(), "one")
	if input, _ := f.ProblemInput(f.ctx(), alice.ID, detail); input != "the input" {
		t.Fatalf("input=%q, want old input", input)
	}
	if ok, err := f.Submit(f.ctx(), SubmitInput{
		UserID: alice.ID, ContestSlug: "alpha", ProblemSlug: "one", Part: 1, Answer: "old-answer",
	}); err != nil || !ok {
		t.Fatalf("old answer: ok=%v err=%v", ok, err)
	}
	_, tasks, _ := f.GenerationRun(f.ctx(), runID)
	if len(tasks) != 1 || tasks[0].Status != "failed" || !strings.Contains(tasks[0].Error, "broken") {
		t.Fatalf("tasks=%+v", tasks)
	}
}

func TestMissingGenerationSkipsCompleteData(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"42"})
	runID, err := f.QueueGeneration(f.ctx(), "alpha", "missing", []int64{alice.ID})
	if err != nil {
		t.Fatalf("queueing: %v", err)
	}
	run, tasks, err := f.GenerationRun(f.ctx(), runID)
	if err != nil || run.Status != "completed" || len(tasks) != 1 || tasks[0].Status != "skipped" {
		t.Fatalf("run=%+v tasks=%+v err=%v", run, tasks, err)
	}
}

func TestGenerationDeduplicatesAndRecoversInterruptedTask(t *testing.T) {
	f, alice := seedProblem(t, 1, []string{"old"})
	f.generator("alpha", "one", "generate_input", "#!/bin/sh\nprintf recovered\n")
	f.generator("alpha", "one", "generate_output1", "#!/bin/sh\nprintf answer\n")
	firstID, err := f.QueueGeneration(f.ctx(), "alpha", "replace", []int64{alice.ID})
	if err != nil {
		t.Fatalf("first queue: %v", err)
	}
	secondID, err := f.QueueGeneration(f.ctx(), "alpha", "replace", []int64{alice.ID})
	if err != nil {
		t.Fatalf("second queue: %v", err)
	}
	second, tasks, err := f.GenerationRun(f.ctx(), secondID)
	if err != nil || second.Status != "completed" || len(tasks) != 1 || tasks[0].Status != "skipped" {
		t.Fatalf("duplicate run=%+v tasks=%+v err=%v", second, tasks, err)
	}
	if _, err := f.Store().ClaimGenerationTask(f.ctx(), sql.NullInt64{Int64: f.clock.Unix(), Valid: true}); err != nil {
		t.Fatalf("claiming before restart: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- f.RunGenerator(ctx, testGenerationConfig()) }()
	defer func() { cancel(); <-done }()
	waitFor(t, func() bool {
		run, _, err := f.GenerationRun(f.ctx(), firstID)
		return err == nil && run.Status == "completed"
	})
	detail, _ := f.Problem(f.ctx(), "one")
	if input, _ := f.ProblemInput(f.ctx(), alice.ID, detail); input != "recovered" {
		t.Fatalf("input=%q, want recovered", input)
	}
}

func TestAllAccountCreationPathsQueueAutomaticGeneration(t *testing.T) {
	f := newFixture(t)
	f.difficulty("facile", 10)
	_, err := f.CreateContest(f.ctx(), CreateContestInput{
		Slug: "alpha", Name: "alpha", StartTime: f.clock.Add(-time.Hour),
		EndTime: f.clock.Add(time.Hour), AutoGenerate: true,
	})
	if err != nil {
		t.Fatalf("contest: %v", err)
	}
	f.problem("alpha", "one", 1)
	if _, err := f.Register(f.ctx(), "admin-created", "admin-created@example.com"); err != nil {
		t.Fatalf("admin registration: %v", err)
	}
	if _, err := f.BatchRegister(f.ctx(), "username,email\nimported,imported@example.com\n"); err != nil {
		t.Fatalf("batch registration: %v", err)
	}
	if _, err := f.EnsureSiteConfig(f.ctx()); err != nil {
		t.Fatalf("site config: %v", err)
	}
	if err := f.UpdateSiteConfig(f.ctx(), sqlc.UpdateSiteConfigParams{
		RegistrationEnabled: sql.NullBool{Bool: true, Valid: true},
	}); err != nil {
		t.Fatalf("enable registration: %v", err)
	}
	if _, err := f.RegisterWithPassword(f.ctx(), "self", "self@example.com", "Passw0rd!"); err != nil {
		t.Fatalf("self registration: %v", err)
	}
	runs, err := f.GenerationRuns(f.ctx(), 10)
	if err != nil || len(runs) != 3 {
		t.Fatalf("runs=%+v err=%v", runs, err)
	}
}

func TestGeneratorScriptLimits(t *testing.T) {
	dir := t.TempDir()
	writeExecutable(t, filepath.Join(dir, "slow"), "#!/bin/sh\nsleep 1\n")
	_, err := runGeneratorScript(context.Background(), GenerationConfig{
		ParallelRuns: 1, Timeout: 10 * time.Millisecond, MaxOutputBytes: 10,
	}, dir, "slow", "")
	if err == nil || !strings.Contains(err.Error(), "délai") {
		t.Fatalf("timeout error=%v", err)
	}
	writeExecutable(t, filepath.Join(dir, "large"), "#!/bin/sh\nprintf 12345678901\n")
	_, err = runGeneratorScript(context.Background(), GenerationConfig{
		ParallelRuns: 1, Timeout: time.Second, MaxOutputBytes: 10,
	}, dir, "large", "")
	if !errors.Is(err, errGeneratorOutputTooLarge) {
		t.Fatalf("large output error=%v", err)
	}
}

func (f *fixture) generator(contest, problem, name, body string) {
	f.t.Helper()
	writeExecutable(f.t, filepath.Join(f.dir, contest, problem, name), body)
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("writing generator: %v", err)
	}
}

func testGenerationConfig() GenerationConfig {
	return GenerationConfig{ParallelRuns: 2, Timeout: time.Second, MaxOutputBytes: 128 << 10}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met")
}
