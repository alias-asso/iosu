package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupProblemDB(t *testing.T) *gorm.DB {

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	err = db.AutoMigrate(
		&database.User{},
		&database.Contest{},
		&database.Difficulty{},
		&database.Problem{},
		&database.Solve{},
	)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func seedContest(t *testing.T, db *gorm.DB) database.Contest {
	c := database.Contest{
		Name:      "round-1",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Hour),
	}
	if err := db.Create(&c).Error; err != nil {
		t.Fatal(err)
	}
	return c
}

func seedDifficulty(t *testing.T, db *gorm.DB) database.Difficulty {
	d := database.Difficulty{Name: "easy", Points: 100}
	if err := db.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	return d
}

func seedProblem(t *testing.T, db *gorm.DB) database.Problem {
	c := seedContest(t, db)
	d := seedDifficulty(t, db)
	p := database.Problem{
		Name:         "Two Sum",
		Slug:         "two-sum",
		Parts:        3,
		ContestID:    c.ID,
		DifficultyID: d.ID,
	}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	return p
}

func seedUser(t *testing.T, db *gorm.DB) database.User {
	u := database.User{
		Username: "alice",
		Email:    "alice@test.com",
		Password: "hash",
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func TestCreateProblem(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)

	c := seedContest(t, db)
	d := seedDifficulty(t, db)

	problem := database.Problem{
		Name:         "Two Sum",
		Slug:         "two-sum",
		ContestID:    c.ID,
		DifficultyID: d.ID,
	}

	err := repo.Create(context.Background(), &problem)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if problem.ID == 0 {
		t.Fatal("expected ID to be set")
	}
}

func TestUpdateProblem(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)
	problem := seedProblem(t, db)

	err := repo.Update(context.Background(), problem.ID, database.Problem{Name: "Updated"})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	var updated database.Problem
	db.First(&updated, problem.ID)

	if updated.Name != "Updated" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}
}

func TestUpdateProblemNotFound(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)

	err := repo.Update(context.Background(), 999, database.Problem{Name: "test"})

	if err != ErrProblemNotFound {
		t.Fatalf("expected ErrProblemNotFound got %v", err)
	}
}

func TestGetBySlug(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)
	problem := seedProblem(t, db)

	result, err := repo.GetBySlug(context.Background(), problem.Slug)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if result.ID != problem.ID {
		t.Fatalf("expected ID %d, got %d", problem.ID, result.ID)
	}
}

func TestGetBySlugNotFound(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)

	_, err := repo.GetBySlug(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateDifficulty(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)

	d := database.Difficulty{Name: "hard", Points: 300}

	err := repo.CreateDifficulty(context.Background(), &d)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if d.ID == 0 {
		t.Fatal("expected ID to be set")
	}
}

func TestGetDifficultyByName(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)
	seedDifficulty(t, db)

	result, err := repo.GetDifficultyByName(context.Background(), "easy")
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if result.Name != "easy" {
		t.Fatalf("expected easy, got %s", result.Name)
	}
}

func TestGetDifficultyByNameNotFound(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)

	_, err := repo.GetDifficultyByName(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetSolveByUserAndProblem(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)
	problem := seedProblem(t, db)
	user := seedUser(t, db)

	solve := database.Solve{
		UserID:    user.ID,
		ProblemID: problem.ID,
		Parts:     2,
		Time:      time.Now(),
	}
	if err := db.Create(&solve).Error; err != nil {
		t.Fatal(err)
	}

	result, err := repo.GetSolveByUserAndProblem(context.Background(), user.ID, problem.ID)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if result.Parts != 2 {
		t.Fatalf("expected 2 parts solved, got %d", result.Parts)
	}
}

func TestGetSolveByUserAndProblemNotFound(t *testing.T) {

	db := setupProblemDB(t)
	repo := NewGormProblemRepository(db)

	_, err := repo.GetSolveByUserAndProblem(context.Background(), 999, 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
