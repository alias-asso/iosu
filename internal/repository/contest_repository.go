package repository

import (
	"context"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
)

type ContestRepository interface {
	Create(ctx context.Context, contest *database.Contest) error
}

type GormContestRepository struct {
	db *gorm.DB
}

func NewGormContest(db *gorm.DB) *GormContestRepository {
	return &GormContestRepository{
		db: db,
	}
}

func (r *GormContestRepository) Create(ctx context.Context, contest *database.Contest) error {
	return gorm.G[database.Contest](r.db).Create(ctx, contest)
}
