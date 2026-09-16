package app

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alias-asso/iosu/internal/store/sqlc"
)

type GenerationConfig struct {
	ParallelRuns   int
	Timeout        time.Duration
	MaxOutputBytes int64
}

func (a *App) QueueGeneration(ctx context.Context, contestSlug, mode string, userIDs []int64) (int64, error) {
	if mode != "replace" && mode != "missing" {
		return 0, ErrInvalidGenerationMode
	}
	if len(userIDs) == 0 {
		return 0, ErrNoUsersSelected
	}
	contest, err := a.Contest(ctx, contestSlug)
	if err != nil {
		return 0, err
	}
	problems, err := a.store.ListProblemsByContest(ctx, contest.ID)
	if err != nil {
		return 0, err
	}
	for _, id := range userIDs {
		if _, err := a.User(ctx, id); err != nil {
			return 0, err
		}
	}

	var runID int64
	err = a.store.Tx(ctx, func(q *sqlc.Queries) error {
		run, err := q.CreateGenerationRun(ctx, sqlc.CreateGenerationRunParams{
			ContestID: contest.ID, Source: "manual", Mode: mode, CreatedAt: a.now().Unix(),
		})
		if err != nil {
			return err
		}
		runID = run.ID
		for _, userID := range userIDs {
			for _, problem := range problems {
				if err := a.createGenerationTask(ctx, q, run.ID, problem.Problem.ID, userID, mode); err != nil {
					return err
				}
			}
		}
		return a.finishGenerationRunIfIdle(ctx, q, run.ID)
	})
	if err != nil {
		return 0, err
	}
	a.wakeGenerator()
	return runID, nil
}

func (a *App) QueueProblemGeneration(ctx context.Context, problemSlug string, userID int64) (int64, error) {
	problem, err := a.Problem(ctx, problemSlug)
	if err != nil {
		return 0, err
	}
	if _, err := a.User(ctx, userID); err != nil {
		return 0, err
	}
	var runID int64
	err = a.store.Tx(ctx, func(q *sqlc.Queries) error {
		run, err := q.CreateGenerationRun(ctx, sqlc.CreateGenerationRunParams{
			ContestID: problem.Contest.ID, Source: "manual", Mode: "replace", CreatedAt: a.now().Unix(),
		})
		if err != nil {
			return err
		}
		runID = run.ID
		if err := a.createGenerationTask(ctx, q, run.ID, problem.Problem.ID, userID, "replace"); err != nil {
			return err
		}
		return a.finishGenerationRunIfIdle(ctx, q, run.ID)
	})
	if err != nil {
		return 0, err
	}
	a.wakeGenerator()
	return runID, nil
}

func (a *App) createGenerationTask(ctx context.Context, q *sqlc.Queries, runID, problemID, userID int64, mode string) error {
	status, message := "queued", ""
	active, err := q.HasActiveGenerationTask(ctx, sqlc.HasActiveGenerationTaskParams{
		ProblemID: problemID, UserID: sql.NullInt64{Int64: userID, Valid: true},
	})
	if err != nil {
		return err
	}
	if active {
		status, message = "skipped", "Une génération est déjà en cours."
	} else if mode == "missing" {
		complete, err := q.HasCompleteProblemData(ctx, sqlc.HasCompleteProblemDataParams{
			ProblemID: problemID, UserID: userID,
		})
		if err != nil {
			return err
		}
		if complete {
			status, message = "skipped", "Les données sont déjà complètes."
		}
	}
	finished := sql.NullInt64{}
	if status == "skipped" {
		finished = sql.NullInt64{Int64: a.now().Unix(), Valid: true}
	}
	_, err = q.CreateGenerationTask(ctx, sqlc.CreateGenerationTaskParams{
		RunID: runID, ProblemID: problemID, UserID: sql.NullInt64{Int64: userID, Valid: true}, Shared: false, Status: status,
		Error: message, CreatedAt: a.now().Unix(), FinishedAt: finished,
	})
	return err
}

func (a *App) QueueFreePlayGeneration(ctx context.Context, contestSlug string) (int64, error) {
	contest, err := a.Contest(ctx, contestSlug)
	if err != nil {
		return 0, err
	}
	if contest.Mode != ContestModeFreePlay {
		return 0, ErrInvalidContestMode
	}
	problems, err := a.store.ListProblemsByContest(ctx, contest.ID)
	if err != nil {
		return 0, err
	}
	problemIDs := make([]int64, len(problems))
	for i, problem := range problems {
		problemIDs[i] = problem.Problem.ID
	}
	return a.queueFreePlayGeneration(ctx, contest.ID, problemIDs)
}

func (a *App) QueueFreePlayProblemGeneration(ctx context.Context, problemSlug string) (int64, error) {
	problem, err := a.Problem(ctx, problemSlug)
	if err != nil {
		return 0, err
	}
	if problem.Contest.Mode != ContestModeFreePlay {
		return 0, ErrInvalidContestMode
	}
	return a.queueFreePlayGeneration(ctx, problem.Contest.ID, []int64{problem.Problem.ID})
}

func (a *App) queueFreePlayGeneration(ctx context.Context, contestID int64, problemIDs []int64) (int64, error) {
	var runID int64
	err := a.store.Tx(ctx, func(q *sqlc.Queries) error {
		run, err := q.CreateGenerationRun(ctx, sqlc.CreateGenerationRunParams{
			ContestID: contestID, Source: "manual", Mode: "replace", CreatedAt: a.now().Unix(),
		})
		if err != nil {
			return err
		}
		runID = run.ID
		for _, problemID := range problemIDs {
			active, err := q.HasActiveSharedGenerationTask(ctx, problemID)
			if err != nil {
				return err
			}
			status, message := "queued", ""
			finished := sql.NullInt64{}
			if active {
				status, message = "skipped", "Une génération est déjà en cours."
				finished = sql.NullInt64{Int64: a.now().Unix(), Valid: true}
			}
			if _, err := q.CreateGenerationTask(ctx, sqlc.CreateGenerationTaskParams{
				RunID: run.ID, ProblemID: problemID, Shared: true,
				Status: status, Error: message, CreatedAt: a.now().Unix(), FinishedAt: finished,
			}); err != nil {
				return err
			}
		}
		return a.finishGenerationRunIfIdle(ctx, q, run.ID)
	})
	if err != nil {
		return 0, err
	}
	a.wakeGenerator()
	return runID, nil
}

func (a *App) enqueueAutomaticGeneration(ctx context.Context, q *sqlc.Queries, userID int64) error {
	contests, err := q.ListAutoGenerateContests(ctx)
	if err != nil {
		return err
	}
	for _, contest := range contests {
		problems, err := q.ListProblemsByContest(ctx, contest.ID)
		if err != nil {
			return err
		}
		if len(problems) == 0 {
			continue
		}
		run, err := q.CreateGenerationRun(ctx, sqlc.CreateGenerationRunParams{
			ContestID: contest.ID, Source: "automatic", Mode: "replace", CreatedAt: a.now().Unix(),
		})
		if err != nil {
			return err
		}
		for _, problem := range problems {
			if err := a.createGenerationTask(ctx, q, run.ID, problem.Problem.ID, userID, "replace"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *App) wakeGenerator() {
	select {
	case a.generationWake <- struct{}{}:
	default:
	}
}

func (a *App) RunGenerator(ctx context.Context, cfg GenerationConfig) error {
	if cfg.ParallelRuns < 1 || cfg.Timeout <= 0 || cfg.MaxOutputBytes < 1 {
		return errors.New("invalid generation configuration")
	}
	if err := a.store.RecoverGenerationTasks(ctx); err != nil {
		return err
	}
	if err := a.store.RecoverGenerationRuns(ctx); err != nil {
		return err
	}
	if err := a.store.FinishIdleGenerationRuns(ctx, sql.NullInt64{Int64: a.now().Unix(), Valid: true}); err != nil {
		return err
	}

	var wg sync.WaitGroup
	for range cfg.ParallelRuns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.generationWorker(ctx, cfg)
		}()
	}
	wg.Wait()
	return ctx.Err()
}

func (a *App) generationWorker(ctx context.Context, cfg GenerationConfig) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		task, err := a.store.ClaimGenerationTask(ctx, sql.NullInt64{Int64: a.now().Unix(), Valid: true})
		if errors.Is(err, sql.ErrNoRows) {
			select {
			case <-ctx.Done():
				return
			case <-a.generationWake:
			case <-ticker.C:
			}
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("claiming generation task: %v", err)
			continue
		}
		a.runGenerationTask(ctx, cfg, task)
	}
}

func (a *App) runGenerationTask(ctx context.Context, cfg GenerationConfig, task sqlc.GenerationTask) {
	now := sql.NullInt64{Int64: a.now().Unix(), Valid: true}
	if err := a.store.MarkGenerationRunStarted(ctx, sqlc.MarkGenerationRunStartedParams{StartedAt: now, ID: task.RunID}); err != nil {
		log.Printf("starting generation run %d: %v", task.RunID, err)
	}
	detail, err := a.store.GetGenerationTaskDetail(ctx, task.ID)
	if err == nil && detail.GenerationRun.Mode == "missing" {
		var complete bool
		complete, err = a.store.HasCompleteProblemData(ctx, sqlc.HasCompleteProblemDataParams{
			ProblemID: detail.Problem.ID, UserID: detail.GenerationTask.UserID.Int64,
		})
		if err == nil && complete {
			a.completeGenerationTask(ctx, task, "skipped", "Les données sont déjà complètes.")
			return
		}
	}
	if err == nil {
		err = a.generateProblemData(ctx, cfg, detail)
	}
	if err != nil {
		log.Printf("generation for task %d: %v", task.ID, err)
		a.completeGenerationTask(ctx, task, "failed", err.Error())
		return
	}
	a.completeGenerationTask(ctx, task, "succeeded", "")
}

func (a *App) completeGenerationTask(ctx context.Context, task sqlc.GenerationTask, status, message string) {
	err := a.store.UpdateGenerationTask(ctx, sqlc.UpdateGenerationTaskParams{
		Status: status, Error: message,
		FinishedAt: sql.NullInt64{Int64: a.now().Unix(), Valid: true}, ID: task.ID,
	})
	if err != nil {
		log.Printf("finishing generation task %d: %v", task.ID, err)
		return
	}
	if err := a.finishGenerationRunIfIdle(ctx, a.store.Queries, task.RunID); err != nil {
		log.Printf("finishing generation run %d: %v", task.RunID, err)
	}
}

func (a *App) finishGenerationRunIfIdle(ctx context.Context, q *sqlc.Queries, runID int64) error {
	counts, err := q.GenerationTaskCounts(ctx, runID)
	if err != nil || counts.Queued+counts.Running != 0 {
		return err
	}
	status := "completed"
	if counts.Failed != 0 {
		status = "completed_with_errors"
	}
	return q.FinishGenerationRun(ctx, sqlc.FinishGenerationRunParams{
		Status: status, FinishedAt: sql.NullInt64{Int64: a.now().Unix(), Valid: true}, ID: runID,
	})
}

func (a *App) generateProblemData(ctx context.Context, cfg GenerationConfig, detail sqlc.GetGenerationTaskDetailRow) error {
	dir := a.problemDir(detail.Contest.Slug, detail.Problem.Slug)
	input, err := runGeneratorScript(ctx, cfg, dir, "generate_input", detail.Username)
	if err != nil {
		return err
	}
	input = strings.TrimSpace(input)
	outputs := make([]string, detail.Problem.Parts)
	for i := int64(1); i <= detail.Problem.Parts; i++ {
		outputs[i-1], err = runGeneratorScript(ctx, cfg, dir, fmt.Sprintf("generate_output%d", i), input)
		if err != nil {
			return err
		}
		outputs[i-1] = strings.TrimSpace(outputs[i-1])
	}
	if detail.GenerationTask.Shared {
		return a.SetFreePlayData(ctx, detail.Problem.Slug, input, outputs)
	}
	return a.SetProblemData(ctx, detail.GenerationTask.UserID.Int64, detail.Problem.Slug, input, outputs)
}

var errGeneratorOutputTooLarge = errors.New("sortie trop volumineuse")

type limitedBuffer struct {
	buffer    bytes.Buffer
	remaining int64
	overflow  bool
	kill      func() error
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if int64(len(p)) > b.remaining {
		if b.remaining > 0 {
			_, _ = b.buffer.Write(p[:b.remaining])
			b.remaining = 0
		}
		b.overflow = true
		if b.kill != nil {
			_ = b.kill()
		}
		return n, nil
	}
	b.remaining -= int64(len(p))
	_, _ = b.buffer.Write(p)
	return n, nil
}

func (b *limitedBuffer) String() string { return b.buffer.String() }

func runGeneratorScript(ctx context.Context, cfg GenerationConfig, dir, name, argument string) (string, error) {
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s n'est pas un fichier exécutable", name)
	}

	commandCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	cmd := exec.CommandContext(commandCtx, "./"+name, argument)
	cmd.Dir = dir
	cmd.WaitDelay = 100 * time.Millisecond
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	stdout := &limitedBuffer{remaining: cfg.MaxOutputBytes}
	stderr := &limitedBuffer{remaining: cfg.MaxOutputBytes}
	stdout.kill = cmd.Cancel
	stderr.kill = stdout.kill
	cmd.Stdout, cmd.Stderr = stdout, stderr
	err = cmd.Run()
	if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("%s a dépassé le délai de %s", name, cfg.Timeout)
	}
	if stdout.overflow || stderr.overflow {
		return "", fmt.Errorf("%s: %w", name, errGeneratorOutputTooLarge)
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return "", fmt.Errorf("%s: %w: %s", name, err, message)
		}
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return stdout.String(), nil
}

func (a *App) GenerationRuns(ctx context.Context, limit int64) ([]GenerationRun, error) {
	if err := a.store.FinishIdleGenerationRuns(ctx, sql.NullInt64{Int64: a.now().Unix(), Valid: true}); err != nil {
		return nil, err
	}
	return a.store.ListGenerationRuns(ctx, limit)
}

func (a *App) GenerationRun(ctx context.Context, id int64) (sqlc.GetGenerationRunRow, []GenerationTask, error) {
	run, err := a.store.GetGenerationRun(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return run, nil, ErrGenerationRunNotFound
	}
	if err != nil {
		return run, nil, err
	}
	tasks, err := a.store.ListGenerationTasksByRun(ctx, id)
	return run, tasks, err
}

func (a *App) CompleteProblemsByUser(ctx context.Context, userID int64) ([]CompleteProblem, error) {
	return a.store.ListCompleteProblemsByUser(ctx, userID)
}

func (a *App) CompleteUsersByProblem(ctx context.Context, problemID int64) ([]User, error) {
	return a.store.ListCompleteUsersByProblem(ctx, problemID)
}
