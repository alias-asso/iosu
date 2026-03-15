package repository

import (
	"context"
	"errors"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
)

var (
	ErrContestNotFound     = errors.New("contest not found")
	ErrContestDoesNotExist = errors.New("this contest does not exist")
)

type ContestRepository interface {
	Create(ctx context.Context, contest *database.Contest) error
	Update(ctx context.Context, id uint, contest database.Contest) error
	GetByName(ctx context.Context, name string) (database.Contest, error)
	Get(ctx context.Context, id uint) (database.Contest, error)
}

type GormContestRepository struct {
	db *gorm.DB
}

func NewGormContestRepository(db *gorm.DB) *GormContestRepository {
	return &GormContestRepository{
		db: db,
	}
}

func (r *GormContestRepository) Create(ctx context.Context, contest *database.Contest) error {
	return gorm.G[database.Contest](r.db).Create(ctx, contest)
}

func (r *GormContestRepository) Update(
	ctx context.Context,
	id uint,
	contest database.Contest,
) error {
	rows, err := gorm.G[database.Contest](r.db).
		Where("id = ?", id).
		Updates(ctx, contest)

	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrContestNotFound
	}

	return nil
}

func (r *GormContestRepository) GetByName(ctx context.Context, name string) (database.Contest, error) {
	return gorm.G[database.Contest](r.db).Select("name = ?", name).First(ctx)
}

func (r *GormContestRepository) Get(ctx context.Context, id uint) (database.Contest, error) {
	return gorm.G[database.Contest](r.db).Select("id = ?", id).First(ctx)
}
