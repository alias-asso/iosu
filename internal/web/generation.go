package web

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/alias-asso/iosu/internal/app"
)

type adminGenerationRun struct {
	ID          int64
	ContestName string
	Source      string
	Mode        string
	Status      string
	CreatedAt   string
	Total       int64
	Queued      int64
	Running     int64
	Succeeded   int64
	Failed      int64
	Skipped     int64
}

type adminGenerationStatus struct {
	Runs   []adminGenerationRun
	Active bool
}

type adminGenerationsPage struct {
	Generation adminGenerationStatus
}

func (s *Server) getAdminGenerations(w http.ResponseWriter, r *http.Request) {
	generation, err := s.adminGenerationStatus(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "admin/generations", adminGenerationsPage{Generation: generation})
}

func (s *Server) adminGenerationStatus(ctx context.Context) (adminGenerationStatus, error) {
	runs, err := s.app.GenerationRuns(ctx, 10)
	if err != nil {
		return adminGenerationStatus{}, err
	}
	status := adminGenerationStatus{Runs: make([]adminGenerationRun, len(runs))}
	for i, run := range runs {
		status.Runs[i] = adminGenerationRun{
			ID: run.ID, ContestName: run.ContestName,
			Source: generationSource(run.Source), Mode: generationMode(run.Mode),
			Status: generationStatus(run.Status), CreatedAt: formatAdminTime(run.CreatedAt),
			Total: run.Total, Queued: run.Queued, Running: run.Running,
			Succeeded: run.Succeeded, Failed: run.Failed, Skipped: run.Skipped,
		}
		if run.Status == "queued" || run.Status == "running" {
			status.Active = true
		}
	}
	return status, nil
}

func (s *Server) postAdminGeneration(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.renderAdminUsers(w, r, "Formulaire invalide.", http.StatusBadRequest)
		return
	}
	ids := make([]int64, 0, len(r.Form["user"]))
	for _, value := range r.Form["user"] {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			s.renderAdminUsers(w, r, "Utilisateur invalide.", http.StatusBadRequest)
			return
		}
		ids = append(ids, id)
	}
	runID, err := s.app.QueueGeneration(r.Context(), r.FormValue("contest"), r.FormValue("mode"), ids)
	if err != nil {
		message, status := describe(err)
		if status == http.StatusInternalServerError {
			s.renderError(w, r, err)
			return
		}
		s.renderAdminUsers(w, r, message, status)
		return
	}
	http.Redirect(w, r, "/admin/generations/"+strconv.FormatInt(runID, 10), http.StatusSeeOther)
}

func (s *Server) getAdminGenerationStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.adminGenerationStatus(r.Context())
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.renderPartial(w, "generation-status", status)
}

type adminGenerationTask struct {
	Username    string
	ProblemName string
	Status      string
	Error       string
}

type adminGenerationRunPage struct {
	Run   adminGenerationRun
	Tasks []adminGenerationTask
}

func (s *Server) getAdminGenerationRun(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("run"), 10, 64)
	if err != nil {
		s.renderError(w, r, app.ErrGenerationRunNotFound)
		return
	}
	run, tasks, err := s.app.GenerationRun(r.Context(), id)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	counts, err := s.app.Store().GenerationTaskCounts(r.Context(), id)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	page := adminGenerationRunPage{Run: adminGenerationRun{
		ID: run.ID, ContestName: run.ContestName, Source: generationSource(run.Source),
		Mode: generationMode(run.Mode), Status: generationStatus(run.Status),
		CreatedAt: formatAdminTime(run.CreatedAt), Total: counts.Total, Queued: counts.Queued,
		Running: counts.Running, Succeeded: counts.Succeeded, Failed: counts.Failed, Skipped: counts.Skipped,
	}, Tasks: make([]adminGenerationTask, len(tasks))}
	for i, task := range tasks {
		page.Tasks[i] = adminGenerationTask{
			Username: task.Username, ProblemName: task.ProblemName,
			Status: generationStatus(task.Status), Error: task.Error,
		}
	}
	s.render(w, r, "admin/generation-run", page)
}

type adminCompleteProblemGroup struct {
	Contest  app.Contest
	Problems []app.CompleteProblem
}

type adminUserProblemsPage struct {
	User   app.User
	Groups []adminCompleteProblemGroup
}

func (s *Server) getAdminUserProblems(w http.ResponseWriter, r *http.Request) {
	user, ok := s.adminUser(w, r)
	if !ok {
		return
	}
	problems, err := s.app.CompleteProblemsByUser(r.Context(), user.ID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	page := adminUserProblemsPage{User: user}
	for _, problem := range problems {
		if len(page.Groups) == 0 || page.Groups[len(page.Groups)-1].Contest.ID != problem.Contest.ID {
			page.Groups = append(page.Groups, adminCompleteProblemGroup{Contest: problem.Contest})
		}
		i := len(page.Groups) - 1
		page.Groups[i].Problems = append(page.Groups[i].Problems, problem)
	}
	s.render(w, r, "admin/user-problems", page)
}

type adminProblemUsersPage struct {
	Problem app.ProblemDetail
	Users   []app.User
}

func (s *Server) getAdminProblemUsers(w http.ResponseWriter, r *http.Request) {
	problem, err := s.app.Problem(r.Context(), r.PathValue("problem"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	users, err := s.app.CompleteUsersByProblem(r.Context(), problem.Problem.ID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	s.render(w, r, "admin/problem-users", adminProblemUsersPage{Problem: problem, Users: users})
}

func (s *Server) postAdminProblemUserGeneration(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.PathValue("user"), 10, 64)
	if err != nil {
		s.renderError(w, r, app.ErrUserNotFound)
		return
	}
	runID, err := s.app.QueueProblemGeneration(r.Context(), r.PathValue("problem"), userID)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/generations/"+strconv.FormatInt(runID, 10), http.StatusSeeOther)
}

func (s *Server) postAdminContestFreePlayGeneration(w http.ResponseWriter, r *http.Request) {
	runID, err := s.app.QueueFreePlayGeneration(r.Context(), r.PathValue("contest"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/generations/"+strconv.FormatInt(runID, 10), http.StatusSeeOther)
}

func (s *Server) postAdminProblemFreePlayGeneration(w http.ResponseWriter, r *http.Request) {
	runID, err := s.app.QueueFreePlayProblemGeneration(r.Context(), r.PathValue("problem"))
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/generations/"+strconv.FormatInt(runID, 10), http.StatusSeeOther)
}

func generationSource(source string) string {
	if source == "automatic" {
		return "Automatique"
	}
	return "Manuelle"
}

func generationMode(mode string) string {
	if mode == "missing" {
		return "Compléter uniquement"
	}
	return "Remplacer"
}

func generationStatus(status string) string {
	switch status {
	case "queued":
		return "En attente"
	case "running":
		return "En cours"
	case "succeeded":
		return "Réussie"
	case "failed":
		return "Échec"
	case "skipped":
		return "Ignorée"
	case "completed_with_errors":
		return "Terminée avec des erreurs"
	default:
		return "Terminée"
	}
}

func formatAdminTime(unix int64) string {
	return time.Unix(unix, 0).Format("02/01/2006 15:04:05")
}
