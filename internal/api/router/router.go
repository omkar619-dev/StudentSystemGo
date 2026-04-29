package router

import "net/http"

func MainRouter() *http.ServeMux {
	tRouter := teachersRouter()
	sRouter := studentsRouter()
	eRouter := execsRouter()

	// /healthz — no auth, no DB. Used for liveness checks and CPU-only benchmarks.
	eRouter.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	sRouter.Handle("/", eRouter)
	tRouter.Handle("/", sRouter)
	return tRouter


	
	// mux.HandleFunc("GET /execs/", handlers.ExecsHandler)

	

}