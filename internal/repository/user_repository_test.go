package repository

import (
	"context"
	"testing"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	err = db.AutoMigrate(
		&database.User{},
		&database.ActivationCode{},
	)

	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestCreateIfNotExist(t *testing.T) {

	db := setupTestDB(t)

	repo := NewGormUserRepository(db)

	ctx := context.Background()

	user := &database.User{
		Username: "user",
		Email:    "user@test.com",
	}

	ok, err := repo.CreateIfNotExist(ctx, user)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("expected user to be created")
	}

	ok, err = repo.CreateIfNotExist(ctx, user)
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("expected duplicate insert to fail")
	}
}

func TestGetByUsername(t *testing.T) {

	db := setupTestDB(t)

	repo := NewGormUserRepository(db)

	ctx := context.Background()

	user := database.User{
		Username: "user",
		Email:    "user@test.com",
	}

	db.Create(&user)

	res, err := repo.GetByUsername(ctx, "user")
	if err != nil {
		t.Fatal(err)
	}

	if res.Username != "user" {
		t.Fatal("wrong user returned")
	}
}

func TestCreateUserWithActivation(t *testing.T) {

	db := setupTestDB(t)

	repo := NewGormUserRepository(db)

	ctx := context.Background()

	user := &database.User{
		Username: "user",
		Email:    "user@test.com",
	}

	activation := &database.ActivationCode{
		Code: "code",
	}

	err := repo.CreateUserWithActivation(ctx, user, activation)
	if err != nil {
		t.Fatal(err)
	}

	var count int64
	db.Model(&database.ActivationCode{}).Count(&count)

	if count != 1 {
		t.Fatal("activation code not created")
	}
}
