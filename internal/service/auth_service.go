package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/mail"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo                 repository.UserRepository
	jwtSecret            string
	defaultAdminPassword string
}

func NewAuthService(repo repository.UserRepository, jwtSecret string, defaultAdminPassword string) AuthService {
	return AuthService{
		repo:                 repo,
		jwtSecret:            jwtSecret,
		defaultAdminPassword: defaultAdminPassword,
	}
}

var (
	ErrUsernameTooLong       = errors.New("username too long")
	ErrUsernameRequired      = errors.New("username required")
	ErrPasswordTooLong       = errors.New("password too long")
	ErrPasswordRequired      = errors.New("password required")
	ErrNonExistantAccount    = errors.New("no account is associated with this username")
	ErrWrongPassword         = errors.New("wrong password")
	ErrInvalidEmail          = errors.New("invalid email")
	ErrUserAlreadyExist      = errors.New("this user already exist")
	ErrInvalidCSV            = errors.New("invalid csv file")
	ErrInvalidCSVHeader      = errors.New("invalid csv header")
	ErrInvalidInput          = errors.New("invalid input")
	ErrUserNotFound          = errors.New("user not found")
	ErrInternalError         = errors.New("internal error")
	ErrInvalidActivationCode = errors.New("activation code is invalid")
	ErrActivationCodeExpired = errors.New("activation code has expired")
	ErrInvalidPassword       = errors.New("Le mot de passe ne respecte pas les critères demandés.")
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	jwt.RegisteredClaims
}

func generateJWT(userID uint, username string, admin bool, jwtSecret string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Admin:    admin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtSecret))

}

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func encryptPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func comparePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func validatePassword(s string) bool {
	if len(s) < 8 {
		return false
	}

	checks := []string{
		`[A-Z]`,                  // uppercase
		`[a-z]`,                  // lowercase
		`[0-9]`,                  // number
		`[!@#$%^&*(),.?":{}|<>]`, // special char
	}

	for _, pattern := range checks {
		matched, _ := regexp.MatchString(pattern, s)
		if !matched {
			return false
		}
	}

	return true
}

type LoginInput struct {
	Username string
	Password string
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (string, error) {
	if input.Username == "" {
		return "", ErrUsernameRequired
	}

	if input.Password == "" {
		return "", ErrPasswordRequired
	}

	user, err := s.repo.GetByUsername(ctx, input.Username)
	if err != nil {
		return "", ErrNonExistantAccount
	}

	if !comparePassword(input.Password, user.Password) {
		return "", ErrWrongPassword
	}

	token, err := generateJWT(user.ID, input.Username, user.Admin, s.jwtSecret)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) CreateDefaultAdmin(ctx context.Context) {

	user := &database.User{
		Username:  "admin",
		Email:     "admin@example.com",
		Admin:     true,
		Activated: true,
	}
	ok, err := s.repo.CreateIfNotExist(ctx, user)
	if err != nil {
		log.Fatalln("Error creating default admin.")
	}

	if ok {
		log.Println("Creating default admin account. Please change password immediately.")
		encryptedPassword, err := encryptPassword(s.defaultAdminPassword)
		if err != nil {
			log.Fatalln("Error encrypting default admin password.")
		}

		s.repo.UpdateByUsername(ctx, database.User{Username: "admin", Password: encryptedPassword})
	}
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) error {
	if input.Username == "" {
		return ErrUsernameRequired
	}

	if input.Password == "" {
		return ErrPasswordRequired
	}

	if len(input.Username) > 16 {
		return ErrUsernameTooLong
	}

	if _, err := mail.ParseAddress(input.Email); err != nil {
		return ErrInvalidEmail
	}

	if len(input.Password) > 64 {
		return ErrPasswordTooLong
	}

	user := database.User{
		Username:  input.Username,
		Email:     input.Email,
		Activated: false,
	}

	activationCode := database.ActivationCode{
		Code:       randSeq(32),
		Expiration: time.Now().Add(time.Hour * 48),
		User:       user,
	}

	err := s.repo.CreateUserWithActivation(ctx, &user, &activationCode)
	if err != nil {
		return ErrUserAlreadyExist
	}
	return nil
}

func (s *AuthService) BatchRegister(ctx context.Context, csvContent string) error {
	reader := csv.NewReader(strings.NewReader(csvContent))

	header, err := reader.Read()
	if err != nil {
		return ErrInvalidCSV
	}

	requiredHeaders := []string{"username", "email"}
	indexes := make([]int, len(requiredHeaders))

	for i, h := range requiredHeaders {
		index := slices.Index(header, h)
		if index == -1 {
			return ErrInvalidCSVHeader
		}
		indexes[i] = index
	}

	lines, err := reader.ReadAll()
	if err != nil {
		return ErrInvalidCSV
	}

	for _, line := range lines {
		username := line[indexes[0]]
		email := line[indexes[1]]

		if username == "" {
			return ErrUsernameRequired
		}

		if len(username) > 16 {
			return ErrUsernameTooLong
		}

		if _, err := mail.ParseAddress(email); err != nil {
			return ErrInvalidEmail
		}

		user := &database.User{
			Username:  username,
			Email:     email,
			Activated: false,
			Admin:     false,
		}

		activation := &database.ActivationCode{
			Code:       randSeq(32),
			Expiration: time.Now().Add(24 * time.Hour),
		}

		err := s.repo.CreateUserWithActivation(ctx, user, activation)
		if err != nil {
			return fmt.Errorf("creating user %s: %w", username, err)
		}
	}

	return nil
}

type GetUserInput struct {
	Username string
}

func (s *AuthService) GetUser(ctx context.Context, input GetUserInput) (database.User, error) {
	user, err := s.repo.GetByUsername(ctx, input.Username)
	if err != nil {
		return database.User{}, ErrUserNotFound
	}
	return user, nil
}

type ActivateInput struct {
	ActivationCode string
	Password       string
}

func (s *AuthService) Activate(ctx context.Context, input ActivateInput) error {
	if len(input.ActivationCode) > 32 {
		return ErrInvalidActivationCode
	}
	activationCode, err := s.repo.GetActivationCode(ctx, input.ActivationCode)
	if err != nil {
		return ErrInvalidActivationCode
	}
	now := time.Now()
	if now.After(activationCode.Expiration) {
		return ErrActivationCodeExpired
	}

	if !validatePassword(input.Password) {
		return ErrInvalidPassword
	}

	user := activationCode.User
	encryptedPassword, err := encryptPassword(input.Password)
	if err != nil {
		return ErrInternalError
	}
	user.Password = encryptedPassword
	user.Activated = true
	return s.repo.UpdateByUsername(ctx, user)
}

type GetActivationCodeInput struct {
	ActivationCode string
}

func (s *AuthService) GetActivationCode(ctx context.Context, input GetActivationCodeInput) (database.ActivationCode, error) {
	if len(input.ActivationCode) > 32 {
		return database.ActivationCode{}, ErrInvalidActivationCode
	}
	activationCode, err := s.repo.GetActivationCode(ctx, input.ActivationCode)
	if err != nil {
		return database.ActivationCode{}, ErrInvalidActivationCode
	}
	now := time.Now()
	if now.After(activationCode.Expiration) {
		return database.ActivationCode{}, ErrActivationCodeExpired
	}

	return activationCode, nil
}

func (s *AuthService) GetActivationCodes(ctx context.Context) ([]database.ActivationCode, error) {
	activationCodes, err := s.repo.GetActivationCodes(ctx)
	if err != nil {
		return []database.ActivationCode{}, ErrInternalError
	}
	return activationCodes, nil
}

func (s *AuthService) GetLeaderboard(ctx context.Context) ([]database.User, error) {
	results, err := s.repo.GetNonAdminUsersWithSolves(ctx)
	if err != nil {
		return []database.User{}, ErrInternalError
	}

	return results, nil
}
