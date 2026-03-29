package api

import (
	"fmt"
	"net/http"
)

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	const tmpl = `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, tmpl, s.cfg.Metrics.FileserverHits())
}

func (s *Server) handleReset(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	s.cfg.Metrics.Reset()
	fmt.Fprintf(w, "Hits: %d", s.cfg.Metrics.FileserverHits())
}
