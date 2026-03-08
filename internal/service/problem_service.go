package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/yuin/goldmark"
)

var (
	ErrDifficultyNotFound = errors.New("difficulty not found")
	ErrProblemNotFound    = errors.New("problem not found")
)

type ProblemService struct {
	repo           repository.ProblemRepository
	contestService *ContestService
	authService    *AuthService
	dataDir        string
}

func NewProblemService(repo repository.ProblemRepository, contestService *ContestService) ProblemService {
	return ProblemService{
		repo:           repo,
		contestService: contestService,
	}
}

type CreateProblemInput struct {
	ContestName      string
	DifficultyName   string
	Name             string
	Slug             string
	PointsMultiplier *float64
	PointsAdder      *uint
	Parts            *uint
}

func (s *ProblemService) CreateProblem(ctx context.Context, input CreateProblemInput) error {
	difficulty, err := s.repo.GetDifficultyByName(ctx, input.DifficultyName)
	if err != nil {
		return ErrDifficultyNotFound
	}

	contest, err := s.contestService.repo.GetByName(ctx, input.ContestName)
	if err != nil {
		return repository.ErrContestNotFound
	}

	problem := database.Problem{
		Name:         input.Name,
		Slug:         input.Slug,
		DifficultyID: difficulty.ID,
		ContestID:    contest.ID,
	}

	if input.Parts != nil {
		problem.Parts = *input.Parts
	}

	if input.PointsAdder != nil {
		problem.PointsAdder = *input.PointsAdder
	}

	if input.PointsMultiplier != nil {
		problem.PointsMultiplier = *input.PointsMultiplier
	}

	problemDirPath := path.Join(s.dataDir, contest.Name, problem.Slug)

	if _, err := os.Stat(problemDirPath); err != nil && errors.Is(err, os.ErrExist) {
		os.Mkdir(problemDirPath, os.ModePerm)
	}

	return nil
}

type GetProblemPartHtmlInput struct {
	Slug   string
	UserID uint
}

func (s *ProblemService) GetProblemPartsHtml(ctx context.Context, input GetProblemPartHtmlInput) ([]string, error) {
	user, err := s.authService.repo.Get(ctx, input.UserID)
	if err != nil {
		return make([]string, 0), ErrUserNotFound
	}
	problem, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return make([]string, 0), ErrProblemNotFound
	}

	for i := range problem.Parts {
		problemPath := path.Join(s.contestService.dataDir, problem.Slug)
		file, err := os.ReadFile(path.Join(problemPath, fmt.Sprintf("part%d.md", i)))
		if err != nil {
			return make([]string, 0), errors.New(fmt.Sprintf("file not found for problem %s part %d", problem.Slug, i))
		}
	}
}
