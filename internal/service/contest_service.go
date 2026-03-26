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
	repo    repository.ContestRepository
	dataDir string
}

func NewConstestService(repo repository.ContestRepository, dataDir string) ContestService {
	return ContestService{
		repo:    repo,
		dataDir: dataDir,
	}
}

type CreateContestInput struct {
	Name      string
	Slug      string
	StartTime time.Time
	EndTime   time.Time
}

func (s *ContestService) CreateContest(ctx context.Context, input CreateContestInput) error {
	if input.Name == "" {
		return ErrEmptyName
	}
	if input.Slug == "" {
		return ErrEmptySlug
	}
	if len(input.Name) >= 20 {
		return ErrNameTooLong
	}

	if len(input.Slug) >= 20 {
		return ErrSlugTooLong
	}

	if input.EndTime.Before(input.StartTime) {
		return ErrInvalidTimeRange
	}

	contest := database.Contest{
		Name:      input.Name,
		Slug:      input.Slug,
		StartTime: input.StartTime,
		EndTime:   input.EndTime,
	}

	err := s.repo.Create(ctx, &contest)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrContestAlreadyExists
		}
		return err
	}

	contestDirPath := path.Join(s.dataDir, input.Slug)

	if info, err := os.Stat(contestDirPath); err == nil && info.IsDir() {
		return ErrDirectoryExists
	}

	os.Mkdir(contestDirPath, os.ModePerm)

	return nil
}

type UpdateContestInput struct {
	ID        uint
	Name      *string
	Slug      *string
	StartTime *time.Time
	EndTime   *time.Time
}

func (s *ContestService) UpdateContest(
	ctx context.Context,
	input UpdateContestInput,
) error {

	update := database.Contest{}

	if input.Name != nil {
		update.Name = *input.Name
	}

	if input.Slug != nil {
		update.Slug = *input.Slug
	}

	if input.StartTime != nil {
		update.StartTime = *input.StartTime
	}

	if input.EndTime != nil {
		update.EndTime = *input.EndTime
	}

	if input.StartTime != nil && input.EndTime != nil {
		if input.EndTime.Before(*input.StartTime) {
			return ErrInvalidTimeRange
		}
	}

	return s.repo.Update(ctx, input.ID, update)
}
