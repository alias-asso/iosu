package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	err = db.AutoMigrate(&database.Contest{})
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestCreateContest(t *testing.T) {

	db := setupDB(t)
	repo := NewGormContestRepository(db)

	contest := database.Contest{
		Name:      "contest",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}

	err := repo.Create(context.Background(), &contest)

	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if contest.ID == 0 {
		t.Fatalf("expected ID to be set")
	}
}

func TestUpdateContest(t *testing.T) {

	db := setupDB(t)
	repo := NewGormContestRepository(db)

	contest := database.Contest{
		Name:      "contest",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}

	err := repo.Create(context.Background(), &contest)
	if err != nil {
		t.Fatal(err)
	}

	update := database.Contest{
		Name: "updated",
	}

	err = repo.Update(context.Background(), contest.ID, update)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var updated database.Contest

	db.First(&updated, contest.ID)

	if updated.Name != "updated" {
		t.Fatalf("expected updated name")
	}
}

func TestUpdateContestNotFound(t *testing.T) {

	db := setupDB(t)
	repo := NewGormContestRepository(db)

	err := repo.Update(context.Background(), 999, database.Contest{
		Name: "test",
	})

	if err != ErrContestNotFound {
		t.Fatalf("expected ErrContestNotFound got %v", err)
	}
}
