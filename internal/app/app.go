// Package app holds the domain logic shared by the web server and the CLI.
// It sits directly on *store.Store; there is no repository indirection.
package app

import (
	"bytes"
	"html/template"
	"regexp"
	"time"

	"github.com/alias-asso/iosu/internal/store"
	"github.com/alias-asso/iosu/internal/store/sqlc"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Rows returned by the store are used as-is rather than copied into a parallel
// set of domain structs.
type (
	User          = sqlc.User
	Contest       = sqlc.Contest
	Problem       = sqlc.Problem
	Difficulty    = sqlc.Difficulty
	SiteConfig    = sqlc.SiteConfig
	ProblemDetail = sqlc.GetProblemBySlugRow
	ProblemInList = sqlc.ListProblemsByContestRow
	Leaderboarder = sqlc.LeaderboardRow
	PendingUser   = sqlc.ListPendingActivationsRow
)

const (
	maxUsernameLen    = 32
	maxEmailLen       = 254
	maxNameLen        = 70
	maxSlugLen        = 70
	maxAuthorLen      = 40
	maxDifficultyLen  = 20
	minPasswordLen    = 8
	maxPasswordLen    = 72 // bcrypt ignores anything past 72 bytes
	activationCodeLen = 32
	activationTTL     = 5 * 24 * time.Hour
)

// 12 is ~250ms per hash.
var bcryptCost = 12

type App struct {
	store   *store.Store
	dataDir string
	now     func() time.Time // swapped in tests
}

func New(s *store.Store, dataDir string) *App {
	return &App{store: s, dataDir: dataDir, now: time.Now}
}

// Store exposes the underlying store for the few callers that legitimately
// need a raw query (the CLI's listing commands).
func (a *App) Store() *store.Store { return a.store }

// slugPattern keeps slugs safe to use as a path segment. Problem and contest
// slugs are joined onto the data directory to find markdown files.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validSlug(s string) bool {
	return s != "" && len(s) <= maxSlugLen && slugPattern.MatchString(s)
}

// contestWindow reports whether the contest is currently open.
func (a *App) contestWindow(c Contest) error {
	now := a.now().Unix()
	switch {
	case now < c.StartAt:
		return ErrContestNotStarted
	case now > c.EndAt:
		return ErrContestFinished
	}
	return nil
}

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

// Markdown renders trusted, admin-authored markdown. Raw HTML is dropped by
// goldmark's default (safe) configuration.
func Markdown(src string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := markdown.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
