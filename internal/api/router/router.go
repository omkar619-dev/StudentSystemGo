package router

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func MainRouter() *http.ServeMux {
	tRouter := teachersRouter()
	sRouter := studentsRouter()
	eRouter := execsRouter()

	// /healthz — no auth, no DB. Used for liveness checks and CPU-only benchmarks.
	eRouter.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// /metrics — Prometheus scrape target. Exposed without auth (private network).
	// In production with public-facing app, restrict via firewall / nginx.
	eRouter.Handle("GET /metrics", promhttp.Handler())

	sRouter.Handle("/", eRouter)
	tRouter.Handle("/", sRouter)
	return tRouter


	
	// mux.HandleFunc("GET /execs/", handlers.ExecsHandler)

	

}