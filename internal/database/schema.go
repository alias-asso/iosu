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
	Name string
}

type Problem struct {
	gorm.Model
	Name         string
	Points       int
	DifficultyID int
	Difficulty   Difficulty
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&User{}, &ActivationCode{}, &Contest{}, &Difficulty{}, &Problem{})
	if err != nil {
		return err
	}
	log.Println("Database migration finished.")
	return nil
}
