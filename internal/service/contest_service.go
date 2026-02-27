package service

import (
	"context"
	"errors"
	"os"
	"path"
	"time"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"gorm.io/gorm"
)

type ContestService struct {
	Repo    repository.ContestRepository
	DataDir string
}

type CreateContestInput struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

var (
	ErrNameTooLong          = errors.New("name too long")
	ErrContestAlreadyExists = errors.New("contest already exists")
	ErrDirectoryExists      = errors.New("directory exists")
	ErrInvalidTimeRange     = errors.New("invalid time range")
)

func (s *ContestService) CreateContest(ctx context.Context, input CreateContestInput) error {
	if len(input.Name) >= 20 {
		return ErrNameTooLong
	}

	if input.EndTime.Before(input.StartTime) {
		return ErrInvalidTimeRange
	}

	contest := database.Contest{
		Name:      input.Name,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	}

	err := s.Repo.Create(ctx, &contest)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrContestAlreadyExists
		}
		return err
	}

	contestDirPath := path.Join(s.DataDir, input.Name)

	if info, err := os.Stat(contestDirPath); err == nil && info.IsDir() {
		return ErrDirectoryExists
	}

	return nil
}
