package repository

import (
	"context"
	"errors"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrProblemNotFound = errors.New("problem not found")
	ErrSolveNotFound   = errors.New("solve not found")
)

type ProblemRepository interface {
	Create(ctx context.Context, problem *database.Problem) error
	Update(ctx context.Context, id uint, problem database.Problem) error
	GetBySlug(ctx context.Context, slug string) (database.Problem, error)
	GetSolveByUserAndProblem(ctx context.Context, userID uint, problemID uint) (database.Solve, error)
	GetAll(ctx context.Context, contestID uint) ([]database.Problem, error)

	CreateDifficulty(ctx context.Context, difficulty *database.Difficulty) error
	GetDifficultyByName(ctx context.Context, name string) (database.Difficulty, error)

	CreateProblemInput(ctx context.Context, problemInput database.ProblemInput) error
	CreateProblemOutput(ctx context.Context, problemOutput database.ProblemOutput) error

	GetProblemInput(ctx context.Context, userID uint, problemID uint) (database.ProblemInput, error)
	GetProblemOutput(ctx context.Context, userID uint, problemID uint, part uint) (database.ProblemOutput, error)

	CreateSolve(ctx context.Context, solve database.Solve) error
	UpdateSolve(ctx context.Context, solveID uint, solve database.Solve) error
	GetSolve(ctx context.Context, userID uint, problemID uint) (database.Solve, error)
}

type GormProblemRepository struct {
	db *gorm.DB
}

func NewGormProblemRepository(db *gorm.DB) *GormProblemRepository {
	return &GormProblemRepository{
		db: db,
	}
}

func (r *GormProblemRepository) Create(ctx context.Context, problem *database.Problem) error {
	return gorm.G[database.Problem](r.db).Create(ctx, problem)
}

func (r *GormProblemRepository) Update(
	ctx context.Context,
	id uint,
	problem database.Problem,
) error {

	rows, err := gorm.G[database.Problem](r.db).
		Where("id = ?", id).
		Updates(ctx, problem)

	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrProblemNotFound
	}

	return nil
}

func (r *GormProblemRepository) GetBySlug(ctx context.Context, slug string) (database.Problem, error) {
	return gorm.G[database.Problem](r.db).Where("slug = ?", slug).First(ctx)
}

func (r *GormProblemRepository) CreateDifficulty(ctx context.Context, difficulty *database.Difficulty) error {
	return gorm.G[database.Difficulty](r.db).Create(ctx, difficulty)
}
func (r *GormProblemRepository) GetDifficultyByName(ctx context.Context, name string) (database.Difficulty, error) {
	return gorm.G[database.Difficulty](r.db).Where("name = ?", name).First(ctx)
}

func (r *GormProblemRepository) GetSolveByUserAndProblem(ctx context.Context, userID uint, problemID uint) (database.Solve, error) {
	return gorm.G[database.Solve](r.db).
		Where("user_id = ? AND problem_id = ?", userID, problemID).
		First(ctx)
}

func (r *GormProblemRepository) CreateProblemInput(ctx context.Context, problemInput database.ProblemInput) error {
	return gorm.G[database.ProblemInput](r.db).Create(ctx, &problemInput)
}

func (r *GormProblemRepository) CreateProblemOutput(ctx context.Context, problemOutput database.ProblemOutput) error {
	return gorm.G[database.ProblemOutput](r.db).Create(ctx, &problemOutput)
}

func (r *GormProblemRepository) GetProblemInput(ctx context.Context, userID uint, problemID uint) (database.ProblemInput, error) {
	return gorm.G[database.ProblemInput](r.db).Where("user_id = ? AND problem_id = ?", userID, problemID).First(ctx)
}

func (r *GormProblemRepository) GetProblemOutput(ctx context.Context, userID uint, problemID uint, part uint) (database.ProblemOutput, error) {
	return gorm.G[database.ProblemOutput](r.db).Where("user_id = ? AND problem_id = ? AND part = ?", userID, problemID, part).First(ctx)
}

func (r *GormProblemRepository) CreateSolve(ctx context.Context, solve database.Solve) error {
	return gorm.G[database.Solve](r.db).Create(ctx, &solve)
}

func (r *GormProblemRepository) UpdateSolve(ctx context.Context, solveID uint, solve database.Solve) error {
	_, err := gorm.G[database.Solve](r.db).Where("id = ?", solveID).Updates(ctx, solve)
	return err
}

func (r *GormProblemRepository) GetSolve(ctx context.Context, userID uint, problemID uint) (database.Solve, error) {
	solve, err := gorm.G[database.Solve](r.db).Where("user_id = ? and problem_id = ?", userID, problemID).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return database.Solve{}, ErrSolveNotFound
		}
		return database.Solve{}, err
	}
	return solve, nil
}

func (r *GormProblemRepository) GetAll(ctx context.Context, contestID uint) ([]database.Problem, error) {
	return gorm.G[database.Problem](r.db).Joins(clause.LeftJoin.Association("Contest"), nil).Find(ctx)
}
