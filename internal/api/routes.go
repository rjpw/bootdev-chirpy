package api

import "net/http"

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /admin/metrics", s.handleMetrics)
	s.mux.HandleFunc("POST /admin/reset", s.handleReset)

	s.mux.Handle(
		"/app/",
		s.cfg.Metrics.MiddlewareMetricsInc(
			http.StripPrefix("/app", http.FileServer(http.Dir("./root"))),
		),
	)
}
