package app

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/alias-asso/iosu/internal/store/sqlc"
)

type ContestStatus string

const (
	ContestUpcoming ContestStatus = "à venir"
	ContestRunning  ContestStatus = "en cours"
	ContestFinished ContestStatus = "terminé"
)

type ArchiveEntry struct {
	Contest Contest
	Status  ContestStatus
}

type CreateContestInput struct {
	Slug      string
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

// CreateContest records a contest and creates its directory under the data
// directory.
func (a *App) CreateContest(ctx context.Context, in CreateContestInput) (Contest, error) {
	if !validSlug(in.Slug) {
		return Contest{}, ErrInvalidSlug
	}
	if in.Name == "" || len(in.Name) > maxNameLen {
		return Contest{}, ErrInvalidName
	}
	if in.EndTime.Before(in.StartTime) {
		return Contest{}, ErrInvalidTimeRange
	}

	contest, err := a.store.CreateContest(ctx, sqlc.CreateContestParams{
		Slug:    in.Slug,
		Name:    in.Name,
		StartAt: in.StartTime.Unix(),
		EndAt:   in.EndTime.Unix(),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Contest{}, ErrContestExists
		}
		return Contest{}, err
	}

	if err := os.MkdirAll(filepath.Join(a.dataDir, contest.Slug), 0o755); err != nil {
		return contest, err
	}
	return contest, nil
}

// Contest looks up a contest by slug.
func (a *App) Contest(ctx context.Context, slug string) (Contest, error) {
	contest, err := a.store.GetContestBySlug(ctx, slug)
	if errors.Is(err, sql.ErrNoRows) {
		return Contest{}, ErrContestNotFound
	}
	return contest, err
}

// Contests lists every contest, most recent first.
func (a *App) Contests(ctx context.Context) ([]Contest, error) {
	return a.store.ListContests(ctx)
}

// UpdateContest applies the non-nil fields of in to an existing contest.
func (a *App) UpdateContest(ctx context.Context, in sqlc.UpdateContestParams) error {
	if in.Slug.Valid && !validSlug(in.Slug.String) {
		return ErrInvalidSlug
	}
	if in.Name.Valid && (in.Name.String == "" || len(in.Name.String) > maxNameLen) {
		return ErrInvalidName
	}
	if in.StartAt.Valid && in.EndAt.Valid && in.EndAt.Int64 < in.StartAt.Int64 {
		return ErrInvalidTimeRange
	}

	n, err := a.store.UpdateContest(ctx, in)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrContestExists
		}
		return err
	}
	if n == 0 {
		return ErrContestNotFound
	}
	return nil
}

func (a *App) Archive(ctx context.Context) ([]ArchiveEntry, error) {
	contests, err := a.store.ListArchivedContests(ctx)
	if err != nil {
		return nil, err
	}
	now := a.now().Unix()
	entries := make([]ArchiveEntry, len(contests))
	for i, c := range contests {
		entries[i] = ArchiveEntry{Contest: c, Status: contestStatus(now, c)}
	}
	return entries, nil
}

func contestStatus(now int64, c Contest) ContestStatus {
	switch {
	case now < c.StartAt:
		return ContestUpcoming
	case now > c.EndAt:
		return ContestFinished
	}
	return ContestRunning
}
