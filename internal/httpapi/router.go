package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rjpw/bootdev-chirpy/internal/application"
)

type ChirpyAPIRouter struct {
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

func NewRouter(environment application.Environment,
	metrics *application.ServerMetrics,
	repositories *application.Repositories,
	staticPath string,
) *ChirpyAPIRouter {
	s := &ChirpyAPIRouter{
		mux:          http.NewServeMux(),
		staticPath:   staticPath,
		environment:  environment,
		Metrics:      metrics,
		Repositories: repositories,
	}
	s.registerRoutes()
	return s
}

func (router *ChirpyAPIRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router.mux.ServeHTTP(w, r)
}

func (router *ChirpyAPIRouter) registerRoutes() {

	router.mux.HandleFunc("GET /admin/metrics", router.handleMetrics)
	router.mux.HandleFunc("POST /admin/reset", router.handleReset)

	// Static assets, with metered access
	router.mux.Handle("/app/",
		router.Metrics.MiddlewareMetricsInc(
			http.StripPrefix("/app",
				http.FileServer(
					http.Dir(router.staticPath)))))

	// Chirps endpoint
	router.mux.HandleFunc("GET /api/chirps", router.handleGetChirps)
	router.mux.HandleFunc("GET /api/chirps/{chirpID}", router.handleGetChirp)
	router.mux.HandleFunc("DELETE /api/chirps/{chirpID}", router.handleDeleteChirp)
	router.mux.HandleFunc("POST /api/chirps",
		withValidBody[ChirpParams](router.handleCreateChirp))

	router.mux.HandleFunc("GET /api/healthz", router.handleHealthz)
	router.mux.HandleFunc("POST /api/login", router.handleLogin)
	router.mux.HandleFunc("POST /api/refresh", router.handleSessionRefresh)
	router.mux.HandleFunc("POST /api/revoke", router.handleSessionRevoke)
	router.mux.HandleFunc("PUT /api/users", router.handleUpdateUser)
	router.mux.HandleFunc("POST /api/users", router.handleCreateUser)

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

func (router *ChirpyAPIRouter) handleReset(w http.ResponseWriter, _ *http.Request) {
	// require server config platform to be "dev" to allow reset,
	// otherwise return 403 Forbidden
	if router.environment.Platform != "dev" {
		respondWithMessage(
			w,
			http.StatusForbidden,
			"Forbidden: reset is only allowed in dev environment",
		)
	} else {
		router.Metrics.Reset()
		err := router.Repositories.Users.DeleteAllUsers(context.Background())
		if err != nil {
			respondWithMessage(w, http.StatusInternalServerError, "Failed to delete all users")
			return
		}
		respondWithMessage(
			w,
			http.StatusOK,
			fmt.Sprintf("Hits: %d", router.Metrics.FileserverHits()),
		)
	}
}

func (router *ChirpyAPIRouter) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (router *ChirpyAPIRouter) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	const tmpl = `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, tmpl, router.Metrics.FileserverHits())
}
