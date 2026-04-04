package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rjpw/bootdev-chirpy/internal/config"
)

type Server struct {
	cfg  *config.Config
	mux  *http.ServeMux
	root string
}

type jsonError struct {
	Error string `json:"error"`
}

func NewServer(cfg *config.Config, root string) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux(), root: root}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) respondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	data, err := json.Marshal(payload)
	if err != nil {
		s.respondWithMessage(w, 500, "unexpected server error")
		return
	}
	fmt.Fprintf(w, "%s", string(data))
}

func (s *Server) respondWithMessage(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%s", message)
}

func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	// require server config platform to be "dev" to allow reset,
	// otherwise return 403 Forbidden
	if s.cfg.Platform != "dev" {
		s.respondWithMessage(w, http.StatusForbidden, "Forbidden: reset is only allowed in dev environment")
	} else {
		s.cfg.Metrics.Reset()
		s.cfg.Users.DeleteAllUsers(context.Background())
		s.respondWithMessage(w, http.StatusOK, fmt.Sprintf("Hits: %d", s.cfg.Metrics.FileserverHits()))
	}
}
