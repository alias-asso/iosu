package server

import (
	"net/http"
)

func (s *Server) getIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	s.render(w, r.Context(), "pages/index.gohtml", nil)
}
