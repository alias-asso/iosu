package server

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"slices"
	"time"

	"github.com/alias-asso/iosu/internal/config"
	"github.com/alias-asso/iosu/internal/database"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Claims struct {
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	jwt.RegisteredClaims
}

func generateJWT(username string, admin bool, config *config.Config) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		Username: username,
		Admin:    admin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(config.JwtKey))

}

func randSeq(n int) string {
	rand.Seed(time.Now().UnixNano())
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func generateActivationCode() string {
	return randSeq(32)
}

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func validateUsername(username string) bool {
	return len(username) <= 16
}

type UserRequest struct {
	username string
	email    string
}

func encryptPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func comparePassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func createDefaultAdmin(db *gorm.DB, config *config.Config, ctx context.Context) {
	_, err := gorm.G[database.User](db).Where("username = ?", "admin").First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		encryptedPassword, err := encryptPassword(config.DefaultAdminPassword)
		if err != nil {
			fmt.Errorf("Error encrypting default admin password.")
		}
		admin_user := database.User{
			Username: "admin",
			Email:    "admin@example.com",
			Password: encryptedPassword,
		}
		err = gorm.G[database.User](db).Create(ctx, &admin_user)

		if err != nil {
			fmt.Errorf("Error inserting default admin user.")
		}
	}
}

// route handler
func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" {
		http.Error(w, "An email is required.", http.StatusBadRequest)
		return
	}

	if password == "" {
		http.Error(w, "A password is required.", http.StatusBadRequest)
		return
	}

	user, err := gorm.G[database.User](s.db).First(r.Context())
	if err != nil {
		http.Error(w, "No account is associated with this email.", http.StatusBadRequest)
		return
	}

	if !comparePassword(password, user.Password) {
		http.Error(w, "Wrong password.", http.StatusBadRequest)
		return
	}

	token, err := generateJWT(username, user.Admin, s.cfg)
	if err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(time.Hour * 4),
	})

	// w.Header().Set("HX-Redirect", "/admin")
	// w.WriteHeader(http.StatusOK)
	// TODO: replace with ok template
	w.Write([]byte("logged in"))
}

// route handler
func (s *Server) postRegisterAccount(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// route handler
func (s *Server) postBatchCreateAccounts(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("accounts")
	if err != nil {
		http.Error(w, "Error retrieving file.", http.StatusUnsupportedMediaType)
		log.Println(err)
		return
	}
	defer file.Close()
	csvReader := csv.NewReader(file)
	csvHeader, err := csvReader.Read()
	if err != nil {
		http.Error(w, "Error reading CSV header.", http.StatusUnsupportedMediaType)
		log.Println(err)
		return
	}
	var headers = [2]string{"username", "email"}
	var indexes = make([]int, len(headers))
	for i, h := range headers {
		index := slices.Index(csvHeader, h)
		if index == -1 {
			http.Error(w, "Invalid CSV header.", http.StatusUnsupportedMediaType)
			return
		}
		indexes[i] = index
	}
	lines, err := csvReader.ReadAll()

	if err != nil {
		http.Error(w, "Error reading CSV.", http.StatusUnsupportedMediaType)
		log.Println(err)
		return
	}
	users := make([]UserRequest, len(lines))
	for i, line := range lines {
		username := line[indexes[0]]
		if !validateUsername(username) {
			http.Error(w, fmt.Sprintf("Invalid username on line %d.", i), http.StatusUnsupportedMediaType)
			return
		}
		email := line[indexes[1]]
		if !validateEmail(email) {
			http.Error(w, fmt.Sprintf("Invalid email on line %d.", i), http.StatusUnsupportedMediaType)
			return
		}
		users[i] = UserRequest{
			username: username,
			email:    email,
		}
	}
	for _, user := range users {
		user := database.User{Username: user.username, Email: user.email, Activated: false, Admin: false}

		activationCode := database.ActivationCode{Code: generateActivationCode(), Expiration: time.Now(), User: user}
		err := gorm.G[database.ActivationCode](s.db).Create(r.Context(), &activationCode)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unable to create account %s.", user.Username), http.StatusUnsupportedMediaType)
			log.Println(err)
			return
		}
	}
	// TODO: replace with ok template
	w.Write([]byte("ok"))
}
