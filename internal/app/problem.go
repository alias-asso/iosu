package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/alias-asso/iosu/internal/store/sqlc"
)

type CreateProblemInput struct {
	ContestSlug      string
	DifficultyName   string
	Slug             string
	Name             string
	Author           string
	Parts            int64
	PointsMultiplier float64
	PointsAdder      int64
}

// CreateProblem records a problem and creates its directory under the contest.
func (a *App) CreateProblem(ctx context.Context, in CreateProblemInput) (Problem, error) {
	if !validSlug(in.Slug) {
		return Problem{}, ErrInvalidSlug
	}
	if in.Name == "" || len(in.Name) > maxNameLen {
		return Problem{}, ErrInvalidName
	}
	if len(in.Author) > maxAuthorLen {
		return Problem{}, fmt.Errorf("%w: author is longer than %d characters", ErrInvalidName, maxAuthorLen)
	}
	if in.Parts < 1 {
		return Problem{}, ErrInvalidPart
	}

	contest, err := a.Contest(ctx, in.ContestSlug)
	if err != nil {
		return Problem{}, err
	}

	difficulty, err := a.store.GetDifficultyByName(ctx, in.DifficultyName)
	if errors.Is(err, sql.ErrNoRows) {
		return Problem{}, ErrDifficultyNotFound
	}
	if err != nil {
		return Problem{}, err
	}

	problem, err := a.store.CreateProblem(ctx, sqlc.CreateProblemParams{
		ContestID:        contest.ID,
		DifficultyID:     difficulty.ID,
		Slug:             in.Slug,
		Name:             in.Name,
		Author:           in.Author,
		Parts:            in.Parts,
		PointsMultiplier: in.PointsMultiplier,
		PointsAdder:      in.PointsAdder,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Problem{}, ErrProblemExists
		}
		return Problem{}, err
	}

	if err := os.MkdirAll(a.problemDir(contest.Slug, problem.Slug), 0o755); err != nil {
		return problem, err
	}
	return problem, nil
}

// UpdateProblem applies the non-nil fields of in to an existing problem.
func (a *App) UpdateProblem(ctx context.Context, in sqlc.UpdateProblemParams) error {
	if in.Slug.Valid && !validSlug(in.Slug.String) {
		return ErrInvalidSlug
	}
	if in.Name.Valid && (in.Name.String == "" || len(in.Name.String) > maxNameLen) {
		return ErrInvalidName
	}
	if in.Parts.Valid && in.Parts.Int64 < 1 {
		return ErrInvalidPart
	}

	n, err := a.store.UpdateProblem(ctx, in)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrProblemExists
		}
		return err
	}
	if n == 0 {
		return ErrProblemNotFound
	}
	return nil
}

// Problem looks up a problem with its contest and difficulty.
func (a *App) Problem(ctx context.Context, slug string) (ProblemDetail, error) {
	if len(slug) > maxSlugLen {
		return ProblemDetail{}, ErrProblemNotFound
	}
	p, err := a.store.GetProblemBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return ProblemDetail{}, ErrProblemNotFound
	}
	return p, err
}

// ProblemIn looks up a problem and checks it belongs to the named contest,
// which is what the URL claims.
func (a *App) ProblemIn(ctx context.Context, contestSlug, problemSlug string) (ProblemDetail, error) {
	p, err := a.Problem(ctx, problemSlug)
	if err != nil {
		return p, err
	}
	if p.Contest.Slug != contestSlug {
		return ProblemDetail{}, ErrProblemNotFound
	}
	return p, nil
}

// Problems lists a contest's problems. It refuses before the contest opens and
// after it closes.
func (a *App) Problems(ctx context.Context, contestSlug string) ([]ProblemInList, error) {
	contest, err := a.Contest(ctx, contestSlug)
	if err != nil {
		return nil, err
	}
	if err := a.contestWindow(contest); err != nil {
		return nil, err
	}
	return a.store.ListProblemsByContest(ctx, contest.ID)
}

// CreateDifficulty adds a difficulty tier.
func (a *App) CreateDifficulty(ctx context.Context, name string, points int64) error {
	if name == "" || len(name) > maxDifficultyLen {
		return ErrInvalidName
	}
	_, err := a.store.CreateDifficulty(ctx, sqlc.CreateDifficultyParams{Name: name, Points: points})
	return err
}

// SolvedParts is how many parts of a problem the user has completed.
func (a *App) SolvedParts(ctx context.Context, userID, problemID int64) (int64, error) {
	return a.store.GetSolvedParts(ctx, sqlc.GetSolvedPartsParams{UserID: userID, ProblemID: problemID})
}

// ProblemStatement renders the markdown for the parts the user has unlocked:
// part N+1 becomes visible once part N is solved.
func (a *App) ProblemStatement(ctx context.Context, userID int64, p ProblemDetail) ([]template.HTML, error) {
	if err := a.contestWindow(p.Contest); err != nil {
		return nil, err
	}
	solved, err := a.SolvedParts(ctx, userID, p.Problem.ID)
	if err != nil {
		return nil, err
	}

	visible := min(solved+1, p.Problem.Parts)
	dir := a.problemDir(p.Contest.Slug, p.Problem.Slug)
	parts := make([]template.HTML, 0, visible)
	for i := int64(1); i <= visible; i++ {
		src, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("part%d.md", i)))
		if err != nil {
			return nil, fmt.Errorf("reading %s part %d: %w", p.Problem.Slug, i, err)
		}
		html, err := Markdown(string(src))
		if err != nil {
			return nil, fmt.Errorf("rendering %s part %d: %w", p.Problem.Slug, i, err)
		}
		parts = append(parts, html)
	}
	return parts, nil
}

// ProblemInput returns the user's personal input for a problem.
func (a *App) ProblemInput(ctx context.Context, userID int64, p ProblemDetail) (string, error) {
	if err := a.contestWindow(p.Contest); err != nil {
		return "", err
	}
	input, err := a.store.GetProblemInput(ctx, sqlc.GetProblemInputParams{
		ProblemID: p.Problem.ID,
		UserID:    userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInputNotFound
	}
	return input, err
}

// SetProblemData stores one user's input and the expected output of every part.
func (a *App) SetProblemData(ctx context.Context, userID int64, problemSlug, input string, outputs []string) error {
	p, err := a.Problem(ctx, problemSlug)
	if err != nil {
		return err
	}
	if int64(len(outputs)) != p.Problem.Parts {
		return fmt.Errorf("%w: got %d, want %d", ErrPartCountMismatch, len(outputs), p.Problem.Parts)
	}

	return a.store.Tx(ctx, func(q *sqlc.Queries) error {
		if err := q.UpsertProblemInput(ctx, sqlc.UpsertProblemInputParams{
			ProblemID: p.Problem.ID,
			UserID:    userID,
			Input:     input,
		}); err != nil {
			return err
		}
		for i, out := range outputs {
			if err := q.UpsertProblemOutput(ctx, sqlc.UpsertProblemOutputParams{
				ProblemID: p.Problem.ID,
				UserID:    userID,
				Part:      int64(i + 1),
				Output:    out,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

type SubmitInput struct {
	UserID      int64
	ContestSlug string
	ProblemSlug string
	Part        int64
	Answer      string
}

// Submit checks an answer. A wrong answer is reported as (false, nil); an error
// means the submission could not be considered at all.
func (a *App) Submit(ctx context.Context, in SubmitInput) (bool, error) {
	p, err := a.ProblemIn(ctx, in.ContestSlug, in.ProblemSlug)
	if err != nil {
		return false, err
	}
	if err := a.contestWindow(p.Contest); err != nil {
		return false, err
	}
	if in.Part < 1 || in.Part > p.Problem.Parts {
		return false, ErrInvalidPart
	}

	solved, err := a.SolvedParts(ctx, in.UserID, p.Problem.ID)
	if err != nil {
		return false, err
	}
	if in.Part <= solved {
		return false, ErrAlreadySolved
	}
	if in.Part > solved+1 {
		return false, ErrInvalidPart
	}

	expected, err := a.store.GetProblemOutput(ctx, sqlc.GetProblemOutputParams{
		ProblemID: p.Problem.ID,
		UserID:    in.UserID,
		Part:      in.Part,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrOutputNotFound
	}
	if err != nil {
		return false, err
	}

	answer := strings.TrimSpace(in.Answer)
	if subtle.ConstantTimeCompare([]byte(answer), []byte(strings.TrimSpace(expected))) != 1 {
		return false, nil
	}

	// UpsertSolve only ever raises the part count, so a replayed or concurrent
	// submission cannot lower it.
	if err := a.store.UpsertSolve(ctx, sqlc.UpsertSolveParams{
		UserID:    in.UserID,
		ProblemID: p.Problem.ID,
		Parts:     in.Part,
		SolvedAt:  a.now().Unix(),
	}); err != nil {
		return false, err
	}
	return true, nil
}

// problemDir is where a problem's markdown and images live.
func (a *App) problemDir(contestSlug, problemSlug string) string {
	return filepath.Join(a.dataDir, contestSlug, problemSlug)
}

// ProblemImage resolves an image path inside a problem directory, rejecting
// anything that escapes the data directory.
func (a *App) ProblemImage(contestSlug, problemSlug, name string) (string, error) {
	if !validSlug(contestSlug) || !validSlug(problemSlug) {
		return "", ErrProblemNotFound
	}
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return "", ErrProblemNotFound
	}

	root, err := filepath.Abs(a.dataDir)
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(filepath.Join(a.problemDir(contestSlug, problemSlug), "img", name))
	if err != nil {
		return "", err
	}
	// Containment is checked against the data directory itself, not against a
	// prefix built from the same untrusted segments.
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", ErrProblemNotFound
	}
	return full, nil
}
