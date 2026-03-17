package server

import (
	"html/template"
	"net/http"
)

func (s *Server) getNotFound(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("views/pages/error.gohtml", "views/layout/base.gohtml", "views/partials/header.gohtml"))
	err := tmpl.Execute(w, "404")
	if err != nil {
		http.Error(w, "internal server error : "+err.Error(), http.StatusInternalServerError)
	}
}
