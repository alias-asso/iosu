package server

import (
	"context"
	"html/template"
	"net/http"
	"path"
)

type LayoutData struct {
	Header HeaderData
	Footer FooterData
	Page   any
}

type HeaderData struct {
	LoggedIn bool
	Page     any
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

	data := LayoutData{
		Header: HeaderData{
			LoggedIn: loggedIn,
			Page:     pageData},
		Footer: FooterData{},
		Page:   pageData,
	}

	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
