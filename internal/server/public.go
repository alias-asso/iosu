package server

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func (s *Server) getNotFound(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("views/pages/error.gohtml", "views/layout/base.gohtml", "views/partials/header.gohtml"))
	err := tmpl.Execute(w, "404")
	if err != nil {
		http.Error(w, "internal server error : "+err.Error(), http.StatusInternalServerError)
	}
}

func parseMarkdown(content string) (template.HTML, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)

	var buf bytes.Buffer
	if err := md.Convert([]byte(content), &buf); err != nil {
		return template.HTML(""), errors.New("failed to parse markdown")
	}
	return template.HTML(buf.String()), nil
}

func (s *Server) getHelp(w http.ResponseWriter, r *http.Request) {
	config, err := s.configService.GetConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	helpHTML, err := parseMarkdown(config.HelpContent)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.renderWithConfig(w, r.Context(), "pages/page.gohtml", helpHTML, config)
}

func (s *Server) getRules(w http.ResponseWriter, r *http.Request) {
	config, err := s.configService.GetConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rulesHTML, err := parseMarkdown(config.RulesContent)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.renderWithConfig(w, r.Context(), "pages/page.gohtml", rulesHTML, config)
}

func (s *Server) getLegal(w http.ResponseWriter, r *http.Request) {
	config, err := s.configService.GetConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	legalHTML, err := parseMarkdown(config.LegalContent)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.renderWithConfig(w, r.Context(), "pages/page.gohtml", legalHTML, config)
}
func (s *Server) getCredits(w http.ResponseWriter, r *http.Request) {
	config, err := s.configService.GetConfig(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	creditsHTML, err := parseMarkdown(config.CreditsContent)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.renderWithConfig(w, r.Context(), "pages/page.gohtml", creditsHTML, config)
}
