package server

import (
	"net/http"
)

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	NotFoundError(w)
}

func (s *Server) unimplementedHandler(w http.ResponseWriter, r *http.Request) {
	LogicError(w, "unimplemented")
}
