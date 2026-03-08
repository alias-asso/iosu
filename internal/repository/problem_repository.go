package repository

import (
	"context"
	"errors"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
)

var ErrProblemNotFound = errors.New("problem not found")

type ProblemRepository interface {
	Create(ctx context.Context, problem *database.Problem) error
	Update(ctx context.Context, id uint, problem database.Problem) error
	GetBySlug(ctx context.Context, slug string) (database.Problem, error)

	CreateDifficulty(ctx context.Context, difficulty *database.Difficulty) error
	GetDifficultyByName(ctx context.Context, name string) (database.Difficulty, error)
}

type GormProblemRepository struct {
	db *gorm.DB
}

func NewGormProblemRepository(db *gorm.DB) *GormProblemRepository {
	return &GormProblemRepository{
		db: db,
	}
}

func (r *GormProblemRepository) Create(ctx context.Context, problem *database.Problem, difficulty *database.Difficulty) error {
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
	return gorm.G[database.Problem](r.db).Select("slug = ?", slug).First(ctx)
}

func (r *GormProblemRepository) CreateDifficulty(ctx context.Context, difficulty *database.Difficulty) error {
	return gorm.G[database.Difficulty](r.db).Create(ctx, difficulty)
}
func (r *GormProblemRepository) GetDifficultyByName(ctx context.Context, name string) (database.Difficulty, error) {
	return gorm.G[database.Difficulty](r.db).Where("name = ?", name).First(ctx)
}
