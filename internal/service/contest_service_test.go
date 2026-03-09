package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alias-asso/iosu/internal/database"
)

type mockContestRepo struct {
	createFn    func(ctx context.Context, contest *database.Contest) error
	updateFn    func(ctx context.Context, id uint, contest database.Contest) error
	getByNameFn func(ctx context.Context, name string) (database.Contest, error)
}

func (m *mockContestRepo) Create(ctx context.Context, contest *database.Contest) error {
	return m.createFn(ctx, contest)
}

func (m *mockContestRepo) Update(ctx context.Context, id uint, contest database.Contest) error {
	return m.updateFn(ctx, id, contest)
}

func (m *mockContestRepo) GetByName(ctx context.Context, name string) (database.Contest, error) {
	if m.getByNameFn != nil {
		return m.getByNameFn(ctx, name)
	}
	return database.Contest{}, nil
}

func TestCreateContest(t *testing.T) {

	start := time.Now()
	end := start.Add(time.Hour)

	tests := []struct {
		name    string
		input   CreateContestInput
		repoErr error
		wantErr error
	}{
		{
			name: "ok",
			input: CreateContestInput{
				Name:      "contest",
				StartTime: start,
				EndTime:   end,
			},
		},
		{
			name: "name too long",
			input: CreateContestInput{
				Name:      "thisnameiswaytoolongfortheservice",
				StartTime: start,
				EndTime:   end,
			},
			wantErr: ErrNameTooLong,
		},
		{
			name: "invalid time",
			input: CreateContestInput{
				Name:      "test",
				StartTime: end,
				EndTime:   start,
			},
			wantErr: ErrInvalidTimeRange,
		},
	}

	for _, tt := range tests {

		repo := &mockContestRepo{
			createFn: func(ctx context.Context, contest *database.Contest) error {
				return tt.repoErr
			},
		}

		service := NewConstestService(repo, "/tmp")

		err := service.CreateContest(context.Background(), tt.input)

		if tt.wantErr == nil && err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
			t.Fatalf("expected %v got %v", tt.wantErr, err)
		}
	}
}

func TestUpdateContest(t *testing.T) {

	start := time.Now()
	end := start.Add(time.Hour)

	tests := []struct {
		name    string
		input   UpdateContestInput
		repoErr error
		wantErr error
	}{
		{
			name: "ok update",
			input: UpdateContestInput{
				ID:        1,
				StartTime: &start,
				EndTime:   &end,
			},
		},
		{
			name: "invalid time",
			input: UpdateContestInput{
				ID:        1,
				StartTime: &end,
				EndTime:   &start,
			},
			wantErr: ErrInvalidTimeRange,
		},
	}

	for _, tt := range tests {

		repo := &mockContestRepo{
			updateFn: func(ctx context.Context, id uint, contest database.Contest) error {
				return tt.repoErr
			},
		}

		service := NewConstestService(repo, "/tmp")

		err := service.UpdateContest(context.Background(), tt.input)

		if tt.wantErr == nil && err != nil {
			t.Fatalf("unexpected error %v", err)
		}

		if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
			t.Fatalf("expected %v got %v", tt.wantErr, err)
		}
	}
}
