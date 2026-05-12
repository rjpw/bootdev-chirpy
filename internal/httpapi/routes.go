package httpapi

import "net/http"

func (s *Server) registerRoutes() {

	s.mux.HandleFunc("GET /api/healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /admin/metrics", s.handleMetrics)
	s.mux.HandleFunc("POST /admin/reset", s.handleReset)

	s.mux.HandleFunc("POST /api/users", s.handleCreateUser)
	s.mux.HandleFunc("POST /api/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/refresh", s.handleSessionRefresh)

	// s.mux.HandleFunc("POST /api/validate_chirp",
	// 	withValidBody[ChirpParams](s.handleValidateChirp))

	s.mux.HandleFunc("POST /api/chirps",
		withValidBody[ChirpParams](s.handleCreateChirp))

	s.mux.HandleFunc("GET /api/chirps", s.handleGetChirps)
	s.mux.HandleFunc("GET /api/chirps/{chirpID}", s.handleGetChirp)

	s.mux.Handle("/app/",
		s.Metrics.MiddlewareMetricsInc(
			http.StripPrefix("/app",
				http.FileServer(
					http.Dir(s.staticPath)))))
}
