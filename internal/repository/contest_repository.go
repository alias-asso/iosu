package repository

import (
	"context"
	"errors"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
)

var ErrContestNotFound = errors.New("contest not found")

type ContestRepository interface {
	Create(ctx context.Context, contest *database.Contest) error
	Update(ctx context.Context, id uint, contest database.Contest) error
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
