package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
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
	_, err := s.authService.repo.Get(ctx, input.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	problem, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, ErrProblemNotFound
	}

	var visibleParts uint = 1
	solve, err := s.repo.GetSolveByUserAndProblem(ctx, input.UserID, problem.ID)
	if err == nil {
		visibleParts = min(solve.Parts+1, problem.Parts)
	}

	problemPath := path.Join(s.contestService.dataDir, problem.Slug)
	result := make([]string, 0, visibleParts)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	for i := range visibleParts {
		filePath := path.Join(problemPath, fmt.Sprintf("part%d.md", i+1))
		file, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("file not found for problem %s part %d", problem.Slug, i+1)
		}

		var buf bytes.Buffer
		if err := md.Convert(file, &buf); err != nil {
			return nil, fmt.Errorf("failed to parse markdown for problem %s part %d", problem.Slug, i+1)
		}
		result = append(result, buf.String())
	}

	return result, nil
}
