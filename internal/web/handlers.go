package web

import (
	"database/sql"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alias-asso/iosu/internal/app"
	"github.com/alias-asso/iosu/internal/store/sqlc"
)

type indexPage struct {
	Contest *app.Contest       // nil when no contest is active
	Archive []app.ArchiveEntry // only loaded when Contest is nil
}

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) {
	config, err := s.app.SiteConfig(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	var page indexPage
	if slug := config.CurrentContest; slug != "" {
		contest, err := s.app.Contest(r.Context(), slug)
		if err != nil {
			// A stale slug must not take the home page down; fall back to
			// the archive as though no contest were active.
			log.Printf("current contest %q: %v", slug, err)
		} else {
			page.Contest = &contest
		}
	}
	if page.Contest == nil {
		if page.Archive, err = s.app.Archive(r.Context()); err != nil {
			s.renderError(w, r, err)
			return
		}
	}
	s.renderWith(w, r, "index", page, config)
}

func (s *Server) getArchive(w http.ResponseWriter, r *http.Request) {
	entries, err := s.app.Archive(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "archive", entries)
}

func (s *Server) markdownPage(pick func(app.SiteConfig) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		config, err := s.app.SiteConfig(r.Context())
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		html, err := app.Markdown(pick(config))
		if err != nil {
			s.renderError(w, r, err)
			return
		}
		s.renderWith(w, r, "page", html, config)
	}
}

type loginPage struct{ Error string }

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFrom(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, r, "login", loginPage{})
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	user, err := s.app.Authenticate(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		msg, status := describe(err)
		w.WriteHeader(status)
		s.render(w, r, "login", loginPage{Error: msg})
		return
	}

	token, err := s.issueToken(user)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.setTokenCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	s.clearTokenCookie(w)
	w.Header().Set("HX-Redirect", "/")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type registerPage struct {
	Username string
	Email    string
	Error    string
	Success  string
}

func (s *Server) getRegister(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFrom(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	config, err := s.app.SiteConfig(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if !config.RegistrationEnabled {
		s.renderError(w, r, app.ErrRegistrationDisabled)
		return
	}
	s.renderWith(w, r, "register", registerPage{}, config)
}

func (s *Server) postRegister(w http.ResponseWriter, r *http.Request) {
	if _, ok := userFrom(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	config, err := s.app.SiteConfig(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if !config.RegistrationEnabled {
		s.renderError(w, r, app.ErrRegistrationDisabled)
		return
	}
	page := registerPage{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
	}
	password := r.FormValue("password")
	if password != r.FormValue("password-confirmation") {
		page.Error = "Les mots de passe ne correspondent pas."
		w.WriteHeader(http.StatusBadRequest)
		s.renderWith(w, r, "register", page, config)
		return
	}
	approved, err := s.app.RegisterWithPassword(r.Context(), page.Username, page.Email, password)
	if err != nil {
		message, status := describe(err)
		if status == http.StatusInternalServerError {
			s.renderError(w, r, err)
			return
		}
		page.Error = message
		w.WriteHeader(status)
		s.renderWith(w, r, "register", page, config)
		return
	}
	if approved {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	page.Username = ""
	page.Email = ""
	page.Success = "Votre inscription a bien été enregistrée. Un administrateur doit maintenant la valider."
	s.renderWith(w, r, "register", page, config)
}

type activatePage struct {
	Error          string
	ActivationCode string
}

func (s *Server) getActivate(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if _, err := s.app.ActivationCode(r.Context(), code); err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "activate-account", activatePage{ActivationCode: code})
}

func (s *Server) postActivate(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("activation-code")
	err := s.app.Activate(r.Context(), code, r.FormValue("password"))
	if err != nil {
		msg, status := describe(err)
		w.WriteHeader(status)
		s.render(w, r, "activate-account", activatePage{Error: msg, ActivationCode: code})
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

type contestPage struct {
	Contest  app.Contest
	Problems []app.ProblemInList
}

func (s *Server) getContest(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	contest, err := s.app.Contest(r.Context(), slug)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	problems, err := s.app.Problems(r.Context(), slug)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "contest", contestPage{Contest: contest, Problems: problems})
}

func (s *Server) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	board, err := s.app.Leaderboard(r.Context(), r.PathValue("slug"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "leaderboard", board)
}

type problemPage struct {
	Problem     app.Problem
	Difficulty  app.Difficulty
	SolvedParts int64
	Content     []template.HTML
}

func (s *Server) getProblem(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)

	detail, err := s.app.ProblemIn(r.Context(), r.PathValue("contest"), r.PathValue("problem"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	solved, err := s.app.SolvedParts(r.Context(), user.ID, detail.Problem.ID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	parts, err := s.app.ProblemStatement(r.Context(), user.ID, detail)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	s.render(w, r, "problem", problemPage{
		Problem:     detail.Problem,
		Difficulty:  detail.Difficulty,
		SolvedParts: solved,
		Content:     parts,
	})
}

func (s *Server) getInput(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)

	detail, err := s.app.ProblemIn(r.Context(), r.PathValue("contest"), r.PathValue("problem"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	input, err := s.app.ProblemInput(r.Context(), user.ID, detail)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\""+detail.Problem.Slug+".txt\"")
	if _, err := w.Write([]byte(input)); err != nil {
		log.Printf("writing input for %s: %v", detail.Problem.Slug, err)
	}
}

func (s *Server) getProblemImage(w http.ResponseWriter, r *http.Request) {
	path, err := s.app.ProblemImage(r.PathValue("contest"), r.PathValue("problem"), r.PathValue("img"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

type responseIndicator struct {
	Error   string
	Success bool
	MaxPart bool
}

func (s *Server) postSubmit(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)

	part, err := strconv.ParseInt(r.PathValue("part"), 10, 64)
	if err != nil {
		s.renderPartial(w, "response-indicator", responseIndicator{Error: "Partie invalide."})
		return
	}

	in := app.SubmitInput{
		UserID:      user.ID,
		ContestSlug: r.PathValue("contest"),
		ProblemSlug: r.PathValue("problem"),
		Part:        part,
		Answer:      r.FormValue("response"),
	}

	correct, err := s.app.Submit(r.Context(), in)
	if err != nil {
		msg, status := describe(err)
		if status == http.StatusInternalServerError {
			log.Printf("submit %s part %d: %v", in.ProblemSlug, part, err)
		}
		s.renderPartial(w, "response-indicator", responseIndicator{Error: msg})
		return
	}
	if !correct {
		s.renderPartial(w, "response-indicator", responseIndicator{})
		return
	}

	detail, err := s.app.Problem(r.Context(), in.ProblemSlug)
	if err != nil {
		// The answer was accepted; only the "is this the last part?" hint is lost.
		s.renderPartial(w, "response-indicator", responseIndicator{Success: true})
		return
	}
	s.renderPartial(w, "response-indicator", responseIndicator{
		Success: true,
		MaxPart: part == detail.Problem.Parts,
	})
}

type adminConfigPage struct {
	Contests []app.Contest
}

type adminUserListItem struct {
	app.User
	ActivationLink string
}

type adminUsersPage struct {
	Users []adminUserListItem
	Error string
}

func (s *Server) getAdminUsers(w http.ResponseWriter, r *http.Request) {
	s.renderAdminUsers(w, r, "", http.StatusOK)
}

func (s *Server) postAdminUsersImport(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("users")
	if err != nil {
		s.renderAdminUsers(w, r, "Sélectionnez un fichier CSV.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	if _, err := s.app.BatchRegister(r.Context(), string(content)); err != nil {
		message, status := describe(err)
		if status == http.StatusInternalServerError {
			s.renderError(w, r, err)
			return
		}
		s.renderAdminUsers(w, r, message, status)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) renderAdminUsers(w http.ResponseWriter, r *http.Request, message string, status int) {
	users, err := s.app.Users(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	page := make([]adminUserListItem, 0, len(users))
	for _, user := range users {
		item := adminUserListItem{User: user.User}
		if user.ActivationCode != "" {
			item.ActivationLink = "/activate/" + user.ActivationCode
		}
		page = append(page, item)
	}
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	s.render(w, r, "admin/users", adminUsersPage{Users: page, Error: message})
}

type adminUserFormPage struct {
	Title    string
	Action   string
	Submit   string
	Username string
	Email    string
	Error    string
}

func (s *Server) getAdminUserNew(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "admin/user-form", adminUserFormPage{
		Title:  "Nouvel utilisateur",
		Action: "/admin/users/new",
		Submit: "Créer l'utilisateur",
	})
}

func (s *Server) postAdminUserNew(w http.ResponseWriter, r *http.Request) {
	page := adminUserFormPage{
		Title:    "Nouvel utilisateur",
		Action:   "/admin/users/new",
		Submit:   "Créer l'utilisateur",
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
	}
	if _, err := s.app.Register(r.Context(), page.Username, page.Email); err != nil {
		s.renderAdminUserFormError(w, r, page, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) getAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	s.render(w, r, "admin/user-form", editUserFormPage(user))
}

func (s *Server) postAdminUserEdit(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	page := adminUserFormPage{
		Title:    "Modifier l'utilisateur",
		Action:   "/admin/users/" + strconv.FormatInt(user.ID, 10) + "/edit",
		Submit:   "Enregistrer",
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
	}
	if err := s.app.UpdateUser(r.Context(), user.ID, page.Username, page.Email); err != nil {
		s.renderAdminUserFormError(w, r, page, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) getAdminUserAdmin(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	s.render(w, r, "admin/user-admin", user)
}

func (s *Server) postAdminUserAdmin(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	if err := s.app.SetAdmin(r.Context(), user.Username, true); err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) postAdminUserApprove(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	if err := s.app.ApproveUser(r.Context(), user.ID); err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) getAdminUserDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	s.render(w, r, "admin/user-delete", user)
}

func (s *Server) postAdminUserDelete(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	if err := s.app.DeleteUser(r.Context(), user.ID); err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) adminUser(w http.ResponseWriter, r *http.Request) (app.User, bool) {
	id, err := strconv.ParseInt(r.PathValue("user"), 10, 64)
	if err != nil {
		s.renderError(w, r, app.ErrUserNotFound)
		return app.User{}, false
	}
	user, err := s.app.User(r.Context(), id)
	if err != nil {
		s.renderError(w, r, err)
		return app.User{}, false
	}
	return user, true
}

func editUserFormPage(user app.User) adminUserFormPage {
	return adminUserFormPage{
		Title:    "Modifier l'utilisateur",
		Action:   "/admin/users/" + strconv.FormatInt(user.ID, 10) + "/edit",
		Submit:   "Enregistrer",
		Username: user.Username,
		Email:    user.Email,
	}
}

func (s *Server) renderAdminUserFormError(w http.ResponseWriter, r *http.Request, page adminUserFormPage, err error) {
	message, status := describe(err)
	if status == http.StatusInternalServerError {
		s.renderError(w, r, err)
		return
	}
	page.Error = message
	w.WriteHeader(status)
	s.render(w, r, "admin/user-form", page)
}

func (s *Server) getAdminContests(w http.ResponseWriter, r *http.Request) {
	contests, err := s.app.Contests(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "admin/contests", contests)
}

const adminContestTimeLayout = "2006-01-02T15:04"

type adminContestFormPage struct {
	Title       string
	Action      string
	Submit      string
	Slug        string
	Name        string
	Description string
	StartAt     string
	EndAt       string
	Unlisted    bool
	Error       string
}

func (s *Server) getAdminContestNew(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, "admin/contest-form", adminContestFormPage{
		Title:  "Nouveau concours",
		Action: "/admin/contests/new",
		Submit: "Créer le concours",
	})
}

func (s *Server) postAdminContestNew(w http.ResponseWriter, r *http.Request) {
	page, startAt, endAt, ok := s.adminContestForm(w, r, "/admin/contests/new")
	if !ok {
		return
	}
	if _, err := s.app.CreateContest(r.Context(), app.CreateContestInput{
		Slug:        page.Slug,
		Name:        page.Name,
		Description: page.Description,
		StartTime:   startAt,
		EndTime:     endAt,
		Unlisted:    page.Unlisted,
	}); err != nil {
		s.renderAdminContestFormError(w, r, page, err)
		return
	}
	http.Redirect(w, r, "/admin/contests", http.StatusSeeOther)
}

func (s *Server) getAdminContestEdit(w http.ResponseWriter, r *http.Request) {
	contest, err := s.app.Contest(r.Context(), r.PathValue("contest"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "admin/contest-form", editContestFormPage(contest))
}

func (s *Server) postAdminContestEdit(w http.ResponseWriter, r *http.Request) {
	contest, err := s.app.Contest(r.Context(), r.PathValue("contest"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	page, startAt, endAt, ok := s.adminContestForm(w, r, "/admin/contests/"+contest.Slug+"/edit")
	if !ok {
		return
	}
	if err := s.app.UpdateContest(r.Context(), sqlc.UpdateContestParams{
		ID:          contest.ID,
		Slug:        sql.NullString{String: page.Slug, Valid: true},
		Name:        sql.NullString{String: page.Name, Valid: true},
		Description: sql.NullString{String: page.Description, Valid: true},
		StartAt:     sql.NullInt64{Int64: startAt.Unix(), Valid: true},
		EndAt:       sql.NullInt64{Int64: endAt.Unix(), Valid: true},
		Unlisted:    sql.NullBool{Bool: page.Unlisted, Valid: true},
	}); err != nil {
		s.renderAdminContestFormError(w, r, page, err)
		return
	}
	http.Redirect(w, r, "/admin/contests", http.StatusSeeOther)
}

func (s *Server) adminContestForm(w http.ResponseWriter, r *http.Request, action string) (adminContestFormPage, time.Time, time.Time, bool) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulaire invalide", http.StatusBadRequest)
		return adminContestFormPage{}, time.Time{}, time.Time{}, false
	}
	page := adminContestFormPage{
		Title:       "Modifier le concours",
		Action:      action,
		Submit:      "Enregistrer",
		Slug:        r.FormValue("slug"),
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		StartAt:     r.FormValue("start-at"),
		EndAt:       r.FormValue("end-at"),
		Unlisted:    r.Form.Has("unlisted"),
	}
	if action == "/admin/contests/new" {
		page.Title = "Nouveau concours"
		page.Submit = "Créer le concours"
	}
	startAt, startErr := time.ParseInLocation(adminContestTimeLayout, page.StartAt, time.Local)
	endAt, endErr := time.ParseInLocation(adminContestTimeLayout, page.EndAt, time.Local)
	if startErr != nil || endErr != nil {
		page.Error = "Date ou heure invalide."
		w.WriteHeader(http.StatusBadRequest)
		s.render(w, r, "admin/contest-form", page)
		return page, time.Time{}, time.Time{}, false
	}
	return page, startAt, endAt, true
}

func (s *Server) renderAdminContestFormError(w http.ResponseWriter, r *http.Request, page adminContestFormPage, err error) {
	message, status := describe(err)
	if status == http.StatusInternalServerError {
		s.renderError(w, r, err)
		return
	}
	page.Error = message
	w.WriteHeader(status)
	s.render(w, r, "admin/contest-form", page)
}

func editContestFormPage(contest app.Contest) adminContestFormPage {
	return adminContestFormPage{
		Title:       "Modifier le concours",
		Action:      "/admin/contests/" + contest.Slug + "/edit",
		Submit:      "Enregistrer",
		Slug:        contest.Slug,
		Name:        contest.Name,
		Description: contest.Description,
		StartAt:     time.Unix(contest.StartAt, 0).Format(adminContestTimeLayout),
		EndAt:       time.Unix(contest.EndAt, 0).Format(adminContestTimeLayout),
		Unlisted:    contest.Unlisted,
	}
}

func (s *Server) getAdminContestDelete(w http.ResponseWriter, r *http.Request) {
	contest, err := s.app.Contest(r.Context(), r.PathValue("contest"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "admin/contest-delete", contest)
}

func (s *Server) postAdminContestDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteContest(r.Context(), r.PathValue("contest")); err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/contests", http.StatusSeeOther)
}

func (s *Server) getAdminConfig(w http.ResponseWriter, r *http.Request) {
	contests, err := s.app.Contests(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "admin/config", adminConfigPage{Contests: contests})
}

func (s *Server) postAdminConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulaire invalide", http.StatusBadRequest)
		return
	}
	field := func(name string) sql.NullString {
		return sql.NullString{String: r.Form.Get(name), Valid: r.Form.Has(name)}
	}
	err := s.app.UpdateSiteConfig(r.Context(), sqlc.UpdateSiteConfigParams{
		SiteTitle:      field("site-title"),
		MainText:       field("main-text"),
		SecondaryText:  field("secondary-text"),
		CurrentContest: field("current-contest"),
		HelpContent:    field("help-content"),
		RulesContent:   field("rules-content"),
		LegalContent:   field("legal-content"),
		CreditsContent: field("credits-content"),
		RegistrationEnabled: sql.NullBool{
			Bool:  r.Form.Has("registration-enabled"),
			Valid: true,
		},
		RegistrationRequiresApproval: sql.NullBool{
			Bool:  r.Form.Has("registration-requires-approval"),
			Valid: true,
		},
	})
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/config", http.StatusSeeOther)
}

type adminDifficultySelector struct {
	Difficulties []app.Difficulty
	Selected     string
	Error        string
}

func (s *Server) getAdminDifficultySelector(w http.ResponseWriter, r *http.Request) {
	s.renderAdminDifficultySelector(w, r, "", "")
}

func (s *Server) postAdminDifficulty(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	points, err := strconv.ParseInt(r.FormValue("points"), 10, 64)
	if err != nil {
		s.renderAdminDifficultySelector(w, r, "", "Nombre de points invalide.")
		return
	}
	if err := s.app.CreateDifficulty(r.Context(), name, points); err != nil {
		msg, status := describe(err)
		if status == http.StatusInternalServerError {
			log.Printf("creating difficulty %q: %v", name, err)
		}
		s.renderAdminDifficultySelector(w, r, "", msg)
		return
	}
	s.renderAdminDifficultySelector(w, r, name, "")
}

func (s *Server) renderAdminDifficultySelector(w http.ResponseWriter, r *http.Request, selected, message string) {
	difficulties, err := s.app.Difficulties(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.renderPartial(w, "admin-difficulty-selector", adminDifficultySelector{
		Difficulties: difficulties,
		Selected:     selected,
		Error:        message,
	})
}
