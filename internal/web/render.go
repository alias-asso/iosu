package web

import (
	"errors"
	"log"
	"net/http"

	"github.com/alias-asso/iosu/internal/app"
)

// layoutData is what every full page template receives.
type layoutData struct {
	LoggedIn bool
	IsAdmin  bool
	Config   app.SiteConfig
	Page     any
}

// render writes a full page. Template and config errors are logged, never
// shown to the visitor.
func (s *Server) render(w http.ResponseWriter, r *http.Request, page string, data any) {
	config, err := s.app.SiteConfig(r.Context())
	if err != nil {
		log.Printf("loading site config: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	s.renderWith(w, r, page, data, config)
}

func (s *Server) renderWith(w http.ResponseWriter, r *http.Request, page string, data any, config app.SiteConfig) {
	tpl, ok := s.templates["pages/"+page]
	if !ok {
		log.Printf("no such template: pages/%s", page)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	user, loggedIn := userFrom(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "base", layoutData{
		LoggedIn: loggedIn,
		IsAdmin:  loggedIn && user.Admin,
		Config:   config,
		Page:     data,
	}); err != nil {
		// The response is likely half-written by now, so only log.
		log.Printf("rendering pages/%s: %v", page, err)
	}
}

// renderPartial writes an htmx fragment.
func (s *Server) renderPartial(w http.ResponseWriter, partial string, data any) {
	tpl, ok := s.templates["partials/"+partial]
	if !ok {
		log.Printf("no such template: partials/%s", partial)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.Execute(w, data); err != nil {
		log.Printf("rendering partials/%s: %v", partial, err)
	}
}

// renderError shows the error page with the French message for err.
func (s *Server) renderError(w http.ResponseWriter, r *http.Request, err error) {
	msg, status := describe(err)
	if status == http.StatusInternalServerError {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	}
	w.WriteHeader(status)
	s.render(w, r, "error", msg)
}

// messages maps domain errors to text shown to visitors. Anything missing is
// reported as a generic internal error, so a new sentinel cannot accidentally
// leak internals.
var messages = map[error]struct {
	text   string
	status int
}{
	app.ErrInvalidCredentials:   {"Nom d'utilisateur ou mot de passe incorrect.", http.StatusUnauthorized},
	app.ErrNotActivated:         {"Ce compte n'est pas encore activé.", http.StatusForbidden},
	app.ErrUserExists:           {"Cet utilisateur existe déjà.", http.StatusConflict},
	app.ErrUserNotFound:         {"Utilisateur introuvable.", http.StatusNotFound},
	app.ErrInvalidCSV:           {"Le fichier CSV doit contenir les colonnes « username » et « email » et des lignes valides.", http.StatusBadRequest},
	app.ErrRegistrationDisabled: {"Les inscriptions sont fermées.", http.StatusNotFound},
	app.ErrUserNeedsPassword:    {"Cet utilisateur doit d'abord choisir un mot de passe.", http.StatusConflict},

	app.ErrInvalidUsername: {"Nom d'utilisateur invalide.", http.StatusBadRequest},
	app.ErrInvalidEmail:    {"Adresse e-mail invalide.", http.StatusBadRequest},
	app.ErrWeakPassword: {"Le mot de passe doit faire au moins 8 caractères et contenir " +
		"une majuscule, une minuscule, un chiffre et un caractère spécial.", http.StatusBadRequest},

	app.ErrInvalidActivationCode: {"Code d'activation invalide.", http.StatusBadRequest},
	app.ErrActivationCodeExpired: {"Ce code d'activation a expiré.", http.StatusBadRequest},
	app.ErrActivationCodeUsed:    {"Ce code d'activation a déjà été utilisé.", http.StatusBadRequest},

	app.ErrContestNotFound:   {"Concours introuvable.", http.StatusNotFound},
	app.ErrContestExists:     {"Un concours utilise déjà cet identifiant.", http.StatusConflict},
	app.ErrContestNotEmpty:   {"Ce concours contient encore des problèmes.", http.StatusConflict},
	app.ErrContestNotStarted: {"Le concours n'a pas encore commencé.", http.StatusForbidden},
	app.ErrContestFinished:   {"Le concours est terminé.", http.StatusForbidden},
	app.ErrInvalidTimeRange:  {"La date de fin doit être postérieure à la date de début.", http.StatusBadRequest},

	app.ErrProblemNotFound:  {"Problème introuvable.", http.StatusNotFound},
	app.ErrDifficultyExists: {"Cette difficulté existe déjà.", http.StatusConflict},
	app.ErrInvalidSlug:      {"Identifiant invalide.", http.StatusBadRequest},
	app.ErrInvalidName:      {"Nom invalide.", http.StatusBadRequest},
	app.ErrInvalidPart:      {"Cette partie n'est pas accessible.", http.StatusBadRequest},
	app.ErrAlreadySolved:    {"Vous avez déjà résolu cette partie.", http.StatusBadRequest},
	app.ErrInputNotFound:    {"Aucun input n'a été généré pour vous sur ce problème.", http.StatusNotFound},
	app.ErrOutputNotFound:   {"Aucune réponse attendue n'est enregistrée pour vous sur ce problème.", http.StatusNotFound},
}

func describe(err error) (string, int) {
	for sentinel, m := range messages {
		if errors.Is(err, sentinel) {
			return m.text, m.status
		}
	}
	return "Erreur interne du serveur.", http.StatusInternalServerError
}
