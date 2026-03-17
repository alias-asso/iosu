package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"gorm.io/gorm"
)

type mockProblemRepo struct {
	createFn                   func(ctx context.Context, problem *database.Problem) error
	updateFn                   func(ctx context.Context, id uint, problem database.Problem) error
	getBySlugFn                func(ctx context.Context, slug string) (database.Problem, error)
	getSolveByUserAndProblemFn func(ctx context.Context, userID uint, problemID uint) (database.Solve, error)

	createDifficultyFn    func(ctx context.Context, difficulty *database.Difficulty) error
	getDifficultyByNameFn func(ctx context.Context, name string) (database.Difficulty, error)

	createProblemInputFn  func(ctx context.Context, problemInput database.ProblemInput) error
	createProblemOutputFn func(ctx context.Context, problemOutput database.ProblemOutput) error

	getProblemInputFn  func(ctx context.Context, userID uint, problemID uint) (database.ProblemInput, error)
	getProblemOutputFn func(ctx context.Context, userID uint, problemID uint, part uint) (database.ProblemOutput, error)

	createSolveFn func(ctx context.Context, solve database.Solve) error
	updateSolveFn func(ctx context.Context, solveID uint, solve database.Solve) error
	getSolveFn    func(ctx context.Context, userID uint, problemID uint) (database.Solve, error)
	getAllFn      func(ctx context.Context, contestID uint) ([]database.Problem, error)
}

func (m *mockProblemRepo) Create(ctx context.Context, problem *database.Problem) error {
	return m.createFn(ctx, problem)
}

func (m *mockProblemRepo) Update(ctx context.Context, id uint, problem database.Problem) error {
	return m.updateFn(ctx, id, problem)
}

func (m *mockProblemRepo) GetBySlug(ctx context.Context, slug string) (database.Problem, error) {
	return m.getBySlugFn(ctx, slug)
}

func (m *mockProblemRepo) GetSolveByUserAndProblem(ctx context.Context, userID uint, problemID uint) (database.Solve, error) {
	return m.getSolveByUserAndProblemFn(ctx, userID, problemID)
}

func (m *mockProblemRepo) CreateDifficulty(ctx context.Context, difficulty *database.Difficulty) error {
	return m.createDifficultyFn(ctx, difficulty)
}

func (m *mockProblemRepo) GetDifficultyByName(ctx context.Context, name string) (database.Difficulty, error) {
	return m.getDifficultyByNameFn(ctx, name)
}

func (m *mockProblemRepo) CreateProblemInput(ctx context.Context, problemInput database.ProblemInput) error {
	return m.createProblemInputFn(ctx, problemInput)
}
func (m *mockProblemRepo) CreateProblemOutput(ctx context.Context, problemOutput database.ProblemOutput) error {
	return m.createProblemOutputFn(ctx, problemOutput)
}

func (m *mockProblemRepo) GetProblemInput(ctx context.Context, userID uint, problemID uint) (database.ProblemInput, error) {
	return m.getProblemInputFn(ctx, userID, problemID)
}
func (m *mockProblemRepo) GetProblemOutput(ctx context.Context, userID uint, problemID uint, part uint) (database.ProblemOutput, error) {
	return m.getProblemOutputFn(ctx, userID, problemID, part)
}

func (m *mockProblemRepo) CreateSolve(ctx context.Context, solve database.Solve) error {
	return m.createSolveFn(ctx, solve)
}
func (m *mockProblemRepo) UpdateSolve(ctx context.Context, solveID uint, solve database.Solve) error {
	return m.updateSolveFn(ctx, solveID, solve)
}
func (m *mockProblemRepo) GetSolve(ctx context.Context, userID uint, problemID uint) (database.Solve, error) {
	return m.getSolveFn(ctx, userID, problemID)
}

func (m *mockProblemRepo) GetAll(ctx context.Context, contestID uint) ([]database.Problem, error) {
	return m.getAllFn(ctx, contestID)
}

// writePartFiles creates part1.md ... partN.md under dataDir/slug/
func writePartFiles(t *testing.T, dataDir string, slug string, n int) {
	t.Helper()
	dir := path.Join(dataDir, slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= n; i++ {
		content := fmt.Sprintf("# Part %d\n\nThis is **part %d** content.", i, i)
		if err := os.WriteFile(path.Join(dir, fmt.Sprintf("part%d.md", i)), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCreateProblemService(t *testing.T) {

	multiplier := 1.5
	adder := uint(10)
	parts := uint(3)

	tests := []struct {
		name          string
		input         CreateProblemInput
		difficultyErr error
		contestErr    error
		wantErr       error
	}{
		{
			name: "ok",
			input: CreateProblemInput{
				ContestName:    "round-1",
				DifficultyName: "easy",
				Name:           "Two Sum",
				Slug:           "two-sum",
			},
		},
		{
			name: "ok with optional fields",
			input: CreateProblemInput{
				ContestName:      "round-1",
				DifficultyName:   "easy",
				Name:             "Two Sum",
				Slug:             "two-sum",
				PointsMultiplier: &multiplier,
				PointsAdder:      &adder,
				Parts:            &parts,
			},
		},
		{
			name: "difficulty not found",
			input: CreateProblemInput{
				ContestName:    "round-1",
				DifficultyName: "nonexistent",
				Name:           "Two Sum",
				Slug:           "two-sum",
			},
			difficultyErr: errors.New("not found"),
			wantErr:       ErrDifficultyNotFound,
		},
		{
			name: "contest not found",
			input: CreateProblemInput{
				ContestName:    "nonexistent",
				DifficultyName: "easy",
				Name:           "Two Sum",
				Slug:           "two-sum",
			},
			contestErr: errors.New("not found"),
			wantErr:    repository.ErrContestNotFound,
		},
	}

	for _, tt := range tests {

		dataDir := t.TempDir()

		problemRepo := &mockProblemRepo{
			getDifficultyByNameFn: func(ctx context.Context, name string) (database.Difficulty, error) {
				if tt.difficultyErr != nil {
					return database.Difficulty{}, tt.difficultyErr
				}
				return database.Difficulty{Model: gorm.Model{ID: 1}, Name: name}, nil
			},
			createFn: func(ctx context.Context, problem *database.Problem) error {
				return nil
			},
		}

		contestRepo := &mockContestRepo{
			createFn: func(ctx context.Context, contest *database.Contest) error {
				return nil
			},
			getByNameFn: func(ctx context.Context, name string) (database.Contest, error) {
				if tt.contestErr != nil {
					return database.Contest{}, tt.contestErr
				}
				return database.Contest{Model: gorm.Model{ID: 1}, Name: name}, nil
			},
		}

		contestService := NewConstestService(contestRepo, dataDir)
		service := NewProblemService(problemRepo, &contestService)

		err := service.CreateProblem(context.Background(), tt.input)

		if tt.wantErr == nil && err != nil {
			t.Fatalf("[%s] unexpected error: %v", tt.name, err)
		}

		if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
			t.Fatalf("[%s] expected %v got %v", tt.name, tt.wantErr, err)
		}
	}
}

func TestGetProblemPartsHtml(t *testing.T) {

	tests := []struct {
		name          string
		slug          string
		totalParts    uint
		solvedParts   uint
		hasSolve      bool
		userErr       error
		problemErr    error
		wantPartCount int
		wantErr       bool
	}{
		{
			name:          "no solve record, only part1 returned",
			slug:          "two-sum",
			totalParts:    3,
			hasSolve:      false,
			wantPartCount: 1,
		},
		{
			name:          "solved 1 part, returns parts 1 and 2",
			slug:          "two-sum",
			totalParts:    3,
			hasSolve:      true,
			solvedParts:   1,
			wantPartCount: 2,
		},
		{
			name:          "solved all parts, capped at problem total",
			slug:          "two-sum",
			totalParts:    3,
			hasSolve:      true,
			solvedParts:   3,
			wantPartCount: 3,
		},
		{
			name:       "user not found",
			slug:       "two-sum",
			totalParts: 3,
			userErr:    errors.New("not found"),
			wantErr:    true,
		},
		{
			name:       "problem not found",
			slug:       "nonexistent",
			totalParts: 3,
			problemErr: errors.New("not found"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {

		dataDir := t.TempDir()

		if !tt.wantErr {
			writePartFiles(t, dataDir, tt.slug, int(tt.totalParts))
		}

		userRepo := &mockUserRepo{
			getFn: func(ctx context.Context, id uint) (database.User, error) {
				if tt.userErr != nil {
					return database.User{}, tt.userErr
				}
				return database.User{Model: gorm.Model{ID: id}}, nil
			},
		}

		problemRepo := &mockProblemRepo{
			getBySlugFn: func(ctx context.Context, slug string) (database.Problem, error) {
				if tt.problemErr != nil {
					return database.Problem{}, tt.problemErr
				}
				return database.Problem{Model: gorm.Model{ID: 1}, Slug: tt.slug, Parts: tt.totalParts}, nil
			},
			getSolveByUserAndProblemFn: func(ctx context.Context, userID uint, problemID uint) (database.Solve, error) {
				if !tt.hasSolve {
					return database.Solve{}, errors.New("not found")
				}
				return database.Solve{Parts: tt.solvedParts}, nil
			},
		}

		contestRepo := &mockContestRepo{
			createFn: func(ctx context.Context, contest *database.Contest) error {
				return nil
			},
		}

		contestService := NewConstestService(contestRepo, dataDir)
		authService := NewAuthService(userRepo, "secret", "adminpass")

		service := NewProblemService(problemRepo, &contestService)
		service.authService = &authService

		result, err := service.GetProblemPartsHtml(context.Background(), GetProblemPartHtmlInput{
			Slug:   tt.slug,
			UserID: 1,
		})

		if tt.wantErr && err == nil {
			t.Fatalf("[%s] expected error, got nil", tt.name)
		}

		if !tt.wantErr && err != nil {
			t.Fatalf("[%s] unexpected error: %v", tt.name, err)
		}

		if !tt.wantErr && len(result) != tt.wantPartCount {
			t.Fatalf("[%s] expected %d parts, got %d", tt.name, tt.wantPartCount, len(result))
		}

		for i, html := range result {
			if len(html) == 0 {
				t.Fatalf("[%s] part %d html is empty", tt.name, i+1)
			}
		}
	}
}
