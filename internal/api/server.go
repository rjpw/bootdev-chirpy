package api

import (
	"net/http"

	"github.com/rjpw/bootdev-chirpy/internal/config"
)

type Server struct {
	cfg  *config.Config
	mux  *http.ServeMux
	root string
}

func NewServer(cfg *config.Config, root string) *Server {
	s := &Server{cfg: cfg, mux: http.NewServeMux(), root: root}
	s.registerRoutes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
