package server

import (
	"context"
	"html/template"
	"net/http"
	"path"

	"github.com/alias-asso/iosu/internal/database"
)

type LayoutData struct {
	LoggedIn bool
	Page     any
	Config   database.Config
}

type FooterData struct{}

func (s *Server) render(w http.ResponseWriter, ctx context.Context, templatePath string, pageData any) {
	tpl := template.Must(template.ParseFiles(
		"views/partials/header.gohtml",
		"views/partials/footer.gohtml",
		"views/layout/base.gohtml",
		path.Join("views", templatePath),
	))

	var loggedIn bool = false
	if ctx.Value("logged_in") != nil {
		loggedIn = ctx.Value("logged_in").(bool)

	}
	config, err := s.configService.GetConfig(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := LayoutData{
		LoggedIn: loggedIn,
		Page:     pageData,
		Config:   config,
	}

	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
