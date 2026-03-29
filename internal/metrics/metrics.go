package metrics

import (
	"log"
	"net/http"
	"sync/atomic"
)

type ServerMetrics struct {
	fileserverHits atomic.Int32
}

func (m *ServerMetrics) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Hits: %d", m.FileserverHits())
		m.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (m *ServerMetrics) FileserverHits() int32 {
	return m.fileserverHits.Load()
}

func (m *ServerMetrics) Reset() {
	m.fileserverHits.Store(0)
}
