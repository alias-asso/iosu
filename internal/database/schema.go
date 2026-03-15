package database

import (
	"log"
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username  string `gorm:"unique"`
	Email     string `gorm:"unique"`
	Password  string
	Activated bool
	Admin     bool
}

type ActivationCode struct {
	gorm.Model
	Code       string `gorm:"unique"`
	Expiration time.Time
	UserID     uint
	User       User
}

type Contest struct {
	gorm.Model
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

type Difficulty struct {
	gorm.Model
	Name   string
	Points uint
}

type Problem struct {
	gorm.Model
	Name             string
	Slug             string
	PointsMultiplier float64 `gorm:"default:1.0"`
	PointsAdder      uint    `gorm:"default:0"`
	Parts            uint    `gorm:"default:1"`
	DifficultyID     uint
	Difficulty       Difficulty
	ContestID        uint
	Contest          Contest
}

type ProblemInput struct {
	gorm.Model
	UserID    uint
	User      User
	ProblemID uint
	Problem   Problem
	Input     string
}

type ProblemOutput struct {
	gorm.Model
	UserID    uint
	User      User
	ProblemID uint
	Problem   Problem
	Part      uint
	Output    string
}

type Solve struct {
	gorm.Model
	UserID    uint
	User      User
	ProblemID uint
	Problem   Problem
	Parts     uint
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&User{}, &ActivationCode{}, &Contest{}, &Difficulty{}, &Problem{})
	if err != nil {
		return err
	}
	log.Println("Database migration finished.")
	return nil
}
