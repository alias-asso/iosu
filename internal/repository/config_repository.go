package repository

import (
	"context"
	"errors"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrConfigNotFound = errors.New("config not found")

type ConfigRepository interface {
	CreateIfNotExist(ctx context.Context, config *database.Config) (bool, error)
	Update(ctx context.Context, config database.Config) error
	Get(ctx context.Context) (database.Config, error)
}

type GormConfigRepository struct {
	db *gorm.DB
}

func NewGormConfigRepository(db *gorm.DB) *GormConfigRepository {
	return &GormConfigRepository{
		db: db,
	}
}

func (r *GormConfigRepository) CreateIfNotExist(ctx context.Context, config *database.Config) (bool, error) {

	config.Singleton = 1

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "singleton"}},
			DoNothing: true,
		}).
		Create(config)

	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

func (r *GormConfigRepository) Update(ctx context.Context, config database.Config) error {

	rows, err := gorm.G[database.Config](r.db).
		Where("singleton = 1").
		Updates(ctx, config)

	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrConfigNotFound
	}

	return nil
}

func (r *GormConfigRepository) Get(ctx context.Context) (database.Config, error) {
	return gorm.G[database.Config](r.db).Where("singleton = 1").First(ctx)
}
