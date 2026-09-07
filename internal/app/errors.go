package app

import "errors"

// Sentinel errors describing what went wrong, in Go's usual style. The web
// layer maps them to user-facing French text; nothing here is displayed
// verbatim.
var (
	ErrInvalidCredentials   = errors.New("invalid username or password")
	ErrNotActivated         = errors.New("account is not activated")
	ErrUserExists           = errors.New("user already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrRegistrationDisabled = errors.New("registration is disabled")
	ErrUserNeedsPassword    = errors.New("user has not chosen a password")

	ErrInvalidUsername = errors.New("invalid username")
	ErrInvalidEmail    = errors.New("invalid email address")
	ErrWeakPassword    = errors.New("password does not meet the requirements")

	ErrInvalidActivationCode = errors.New("invalid activation code")
	ErrActivationCodeExpired = errors.New("activation code has expired")
	ErrActivationCodeUsed    = errors.New("activation code has already been used")

	ErrInvalidCSV = errors.New("invalid CSV")

	ErrContestNotFound   = errors.New("contest not found")
	ErrContestExists     = errors.New("contest already exists")
	ErrContestNotEmpty   = errors.New("contest contains problems")
	ErrContestNotStarted = errors.New("contest has not started")
	ErrContestFinished   = errors.New("contest is over")
	ErrInvalidTimeRange  = errors.New("end time is before start time")

	ErrProblemNotFound    = errors.New("problem not found")
	ErrProblemExists      = errors.New("problem already exists")
	ErrDifficultyNotFound = errors.New("difficulty not found")
	ErrDifficultyExists   = errors.New("difficulty already exists")
	ErrInvalidSlug        = errors.New("invalid slug")
	ErrInvalidName        = errors.New("invalid name")
	ErrInvalidPart        = errors.New("invalid part")
	ErrAlreadySolved      = errors.New("part already solved")
	ErrInputNotFound      = errors.New("no input generated for this user")
	ErrOutputNotFound     = errors.New("no expected output for this user")
	ErrPartCountMismatch  = errors.New("number of outputs does not match the problem's part count")
)
