package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path"
	"time"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

type ProblemService struct {
	repo           repository.ProblemRepository
	contestService *ContestService
	authService    *AuthService
	dataDir        string
}

func NewProblemService(repo repository.ProblemRepository, contestService *ContestService, authService *AuthService, dataDir string) ProblemService {
	return ProblemService{
		repo:           repo,
		contestService: contestService,
		authService:    authService,
		dataDir:        dataDir,
	}
}

type CreateProblemInput struct {
	ContestName      string
	DifficultyName   string
	Name             string
	Slug             string
	Author           string
	PointsMultiplier *float64
	PointsAdder      *uint
	Parts            *uint
}

func (s *ProblemService) CreateProblem(ctx context.Context, input CreateProblemInput) error {
	if len(input.Name) >= 20 {
		return ErrNameTooLong
	}

	if len(input.Slug) >= 20 {
		return ErrSlugTooLong
	}

	if len(input.Author) >= 40 {
		return ErrSlugTooLong
	}
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
		Author:       input.Author,
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

	problemDirPath := path.Join(s.dataDir, contest.Slug, problem.Slug)

	if _, err := os.Stat(problemDirPath); err != nil && errors.Is(err, os.ErrExist) {
		os.Mkdir(problemDirPath, os.ModePerm)
	}

	err = s.repo.Create(ctx, &problem)
	if err != nil {
		return ErrUnableToCreateProblem
	}

	return nil
}

type GetProblemPartHtmlInput struct {
	Slug   string
	UserID uint
}

func (s *ProblemService) GetProblemPartsHtml(ctx context.Context, input GetProblemPartHtmlInput) ([]template.HTML, error) {
	_, err := s.authService.repo.Get(ctx, input.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	problem, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return nil, ErrProblemNotFound
	}

	userSolves, err := s.repo.GetSolvesAmount(ctx, input.UserID, problem.ID)
	if err != nil {
		return nil, ErrInternalError
	}
	visibleParts := min(userSolves+1, problem.Parts)

	problemPath := path.Join(s.contestService.dataDir, problem.Contest.Name, problem.Slug)
	result := make([]template.HTML, visibleParts)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	for i := 0; i < int(visibleParts); i++ {
		filePath := path.Join(problemPath, fmt.Sprintf("part%d.md", i+1))
		file, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("file not found for problem %s part %d", problem.Slug, i+1)
		}

		var buf bytes.Buffer
		if err := md.Convert(file, &buf); err != nil {
			return nil, fmt.Errorf("failed to parse markdown for problem %s part %d", problem.Slug, i+1)
		}
		result[i] = template.HTML(buf.String())
	}

	return result, nil
}

type CreateProblemDataInput struct {
	UserID       uint
	Slug         string
	InputValue   string
	OutputValues []string
}

func (s *ProblemService) CreateProblemData(ctx context.Context, input CreateProblemDataInput) error {
	problem, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return ErrProblemNotFound
	}
	if uint(len(input.OutputValues)) != problem.Parts {
		return ErrPartTooBig
	}
	user, err := s.authService.repo.Get(ctx, input.UserID)
	if err != nil {
		return ErrUserNotFound
	}

	for i, out := range input.OutputValues {
		problemOutput := database.ProblemOutput{
			UserID:    user.ID,
			ProblemID: problem.ID,
			Part:      uint(i + 1),
			Output:    out,
		}
		err := s.repo.CreateProblemOutput(ctx, problemOutput)
		if err != nil {
			return ErrCreatingData
		}
	}

	problemInput := database.ProblemInput{
		UserID:    user.ID,
		ProblemID: problem.ID,
		Input:     input.InputValue,
	}

	err = s.repo.CreateProblemInput(ctx, problemInput)
	if err != nil {
		return ErrCreatingData
	}

	return nil
}

type SubmitInput struct {
	UserID      uint
	Slug        string
	ContestSlug string
	Value       string
	Part        uint
}

func (s *ProblemService) Submit(ctx context.Context, input SubmitInput) (bool, error) {
	// Get problem + user + contest
	problem, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return false, ErrProblemNotFound
	}
	if input.Part > problem.Parts {
		return false, ErrPartTooBig
	}
	if problem.Contest.Slug != input.ContestSlug {
		return false, ErrContestNotFound
	}

	solves, err := s.repo.GetSolvesAmount(ctx, input.UserID, problem.ID)
	if solves >= input.Part {
		return false, ErrAlreadySolved
	}
	if input.Part > solves+1 {
		return false, ErrPartTooBig
	}

	user, err := s.authService.repo.Get(ctx, input.UserID)
	if err != nil {
		return false, ErrUserNotFound
	}

	contest := problem.Contest

	// Check if contest not finished/started
	currentTime := time.Now()
	if currentTime.Before(contest.StartTime) {
		return false, ErrContestNotStarted
	}
	if currentTime.After(contest.EndTime) {
		return false, ErrContestFinished
	}

	// Get output
	outputData, err := s.repo.GetProblemOutput(ctx, user.ID, problem.ID, input.Part)
	if err != nil {
		return false, ErrOutputNotFound
	}

	// Check submited value
	if outputData.Output != input.Value {
		return false, nil
	}

	// Create solve
	solve := database.Solve{
		UserID:    user.ID,
		ProblemID: problem.ID,
		Parts:     input.Part,
	}

	// First part -> create solve
	if input.Part <= 1 {
		err := s.repo.CreateSolve(ctx, solve)
		if err != nil {
			return false, ErrUnableToSolve
		}
		return true, nil
	}

	// Else update solve
	previousSolve, err := s.repo.GetSolve(ctx, user.ID, problem.ID)
	if err != nil {
		return false, ErrUnableToSolve
	}

	err = s.repo.UpdateSolve(ctx, previousSolve.ID, solve)
	if err != nil {
		return false, ErrUnableToSolve
	}
	return true, nil
}

type GetProblemsInput struct {
	ContestSlug string
}

func (s *ProblemService) GetProblems(ctx context.Context, input GetProblemsInput) ([]database.Problem, error) {
	contest, err := s.contestService.repo.GetByName(ctx, input.ContestSlug)
	if err != nil {
		return []database.Problem{}, ErrContestNotFound
	}
	problems, err := s.repo.GetAll(ctx, contest.ID)
	if err != nil {
		return []database.Problem{}, ErrProblemNotFound
	}
	return problems, nil
}

type CreateDifficultyInput struct {
	DifficultyName string
	Points         uint
}

func (s *ProblemService) CreateDifficulty(ctx context.Context, input CreateDifficultyInput) error {
	if len(input.DifficultyName) > 20 {
		return ErrNameTooLong
	}
	difficulty := database.Difficulty{
		Name:   input.DifficultyName,
		Points: input.Points,
	}
	err := s.repo.CreateDifficulty(ctx, &difficulty)
	if err != nil {
		return ErrCreatingData
	}
	return nil
}

type GetProblemInput struct {
	Slug string
}

func (s *ProblemService) GetProblem(ctx context.Context, input GetProblemInput) (database.Problem, error) {
	if len(input.Slug) > 20 {
		return database.Problem{}, ErrNameTooLong
	}
	problem, err := s.repo.GetBySlug(ctx, input.Slug)
	if err != nil {
		return database.Problem{}, ErrProblemNotFound
	}
	return problem, nil
}

type GetSolvesInput struct {
	UserID    uint
	ProblemID uint
}

func (s *ProblemService) GetSolves(ctx context.Context, input GetSolvesInput) (uint, error) {
	return s.repo.GetSolvesAmount(ctx, input.UserID, input.ProblemID)
}

// TODO: maybe change this name (confusing ?)
type GetProblemInputInput struct {
	UserID    uint
	ProblemID uint
}

// same thing here
func (s *ProblemService) GetProblemInput(ctx context.Context, input GetProblemInputInput) (string, error) {
	problemInput, err := s.repo.GetProblemInput(ctx, input.UserID, input.ProblemID)
	if err != nil {
		return "", ErrInputNotFound
	}
	return problemInput.Input, nil
}

type UpdateProblemInput struct {
	ID               uint
	Slug             *string
	Name             *string
	Author           *string
	PointsMultiplier *float64
	PointsAdder      *uint
	Parts            *uint
}

func (s *ProblemService) UpdateProblem(
	ctx context.Context,
	input UpdateProblemInput,
) error {
	update := database.Problem{}

	if input.Name != nil {
		update.Name = *input.Name
	}
	if input.Slug != nil {
		update.Slug = *input.Slug
	}
	if input.Author != nil {
		update.Author = *input.Author
	}
	if input.PointsMultiplier != nil {
		update.PointsMultiplier = *input.PointsMultiplier
	}
	if input.PointsAdder != nil {
		update.PointsAdder = *input.PointsAdder
	}
	if input.Parts != nil {
		update.Parts = *input.Parts
	}

	return s.repo.Update(ctx, input.ID, update)
}
