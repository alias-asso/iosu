package server

import (
	"html/template"
	"net/http"
	"path"
)

type LayoutData struct {
	Header HeaderData
	Footer FooterData
	Page   any
}

type HeaderData struct{}

type FooterData struct{}

func (s *Server) render(w http.ResponseWriter, templatePath string, pageData any) {
	tpl := template.Must(template.ParseFiles(
		"views/partials/header.html",
		"views/partials/footer.html",
		"views/layout/base.gohtml",
		path.Join("views", templatePath),
	))

	data := LayoutData{
		Header: HeaderData{},
		Footer: FooterData{},
		Page:   pageData,
	}

	if err := tpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
