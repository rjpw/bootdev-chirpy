package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rjpw/bootdev-chirpy/internal/application"
)

type Server struct {
	mux          *http.ServeMux
	staticPath   string
	environment  application.Environment
	Platform     string
	Metrics      *application.ServerMetrics
	Repositories *application.Repositories
}

type jsonError struct {
	Error string `json:"error"`
}

func NewServer(environment application.Environment,
	metrics *application.ServerMetrics,
	repositories *application.Repositories,
	staticPath string) *Server {
	s := &Server{
		mux:          http.NewServeMux(),
		staticPath:   staticPath,
		environment:  environment,
		Metrics:      metrics,
		Repositories: repositories,
	}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	data, err := json.Marshal(payload)
	if err != nil {
		respondWithMessage(w, 500, "unexpected server error")
		return
	}
	fmt.Fprintf(w, "%s", string(data))
}

func respondWithMessage(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, "%s", message)
}

func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	// require server config platform to be "dev" to allow reset,
	// otherwise return 403 Forbidden
	if s.environment.Platform != "dev" {
		respondWithMessage(
			w,
			http.StatusForbidden,
			"Forbidden: reset is only allowed in dev environment",
		)
	} else {
		s.Metrics.Reset()
		err := s.Repositories.Users.DeleteAllUsers(context.Background())
		if err != nil {
			respondWithMessage(w, http.StatusInternalServerError, "Failed to delete all users")
			return
		}
		respondWithMessage(
			w,
			http.StatusOK,
			fmt.Sprintf("Hits: %d", s.Metrics.FileserverHits()),
		)
	}
}
