package repository

import (
	"context"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository interface {
	UpdateByUsername(ctx context.Context, user database.User) error
	CreateIfNotExist(ctx context.Context, user *database.User) (bool, error)
	Get(ctx context.Context, userID uint) (database.User, error)
	GetByUsername(ctx context.Context, username string) (database.User, error)
	CreateUserWithActivation(ctx context.Context, user *database.User, activation *database.ActivationCode) error
	GetActivationCode(ctx context.Context, code string) (database.ActivationCode, error)
	GetActivationCodes(ctx context.Context) ([]database.ActivationCode, error)
}

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{
		db: db,
	}
}

func (r *GormUserRepository) CreateIfNotExist(ctx context.Context, user *database.User) (bool, error) {
	res := gorm.WithResult()
	err := gorm.G[database.User](r.db,
		clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoNothing: true,
		},
		res,
	).Create(ctx, user)
	return res.RowsAffected == 1, err
}

func (r *GormUserRepository) UpdateByUsername(ctx context.Context, user database.User) error {
	_, err := gorm.G[database.User](r.db).Where("username = ?", user.Username).Updates(ctx, user)
	return err
}

func (r *GormUserRepository) GetByUsername(ctx context.Context, username string) (database.User, error) {
	return gorm.G[database.User](r.db).Where("username = ?", username).First(ctx)
}

func (r *GormUserRepository) Get(ctx context.Context, userID uint) (database.User, error) {
	return gorm.G[database.User](r.db).Where("id = ?", userID).First(ctx)
}

func (r *GormUserRepository) CreateUserWithActivation(
	ctx context.Context,
	user *database.User,
	activation *database.ActivationCode,
) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if err := gorm.G[database.User](tx).Create(ctx, user); err != nil {
			return err
		}

		activation.UserID = user.ID

		if err := gorm.G[database.ActivationCode](tx).Create(ctx, activation); err != nil {
			return err
		}

		return nil
	})
}

func (r *GormUserRepository) GetActivationCode(ctx context.Context, code string) (database.ActivationCode, error) {
	return gorm.G[database.ActivationCode](r.db).Preload("User", nil).Where("code = ?", code).First(ctx)
}

func (r *GormUserRepository) GetActivationCodes(ctx context.Context) ([]database.ActivationCode, error) {
	return gorm.G[database.ActivationCode](r.db).
		Joins(clause.InnerJoin.Association("User"), nil).
		Where("users.activated = ?", true).
		Preload("User", nil).
		Find(ctx)
}
