package debug

import (
	"net/http"
	"sync"
	"sync/atomic"

	_ "net/http/pprof"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func StartDebugServer(wg *sync.WaitGroup) (*http.Server, *atomic.Value) {
	srv := &http.Server{Addr: ":6060"}
	isReady := &atomic.Value{}
	isReady.Store(false)

	// Add profiling server for live profile of the program.
	http.Handle("/metrics", promhttp.Handler())
	// Readines and liveness endpoints.
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/readyz", readyz(isReady))
	// pprof endpoints are added through the import.

	go func() {
		defer wg.Done() // Let main know we are done cleaning up

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Error on sidecar server for debugging")
		}
	}()

	return srv, isReady
}

// healthz is a liveness probe.
func healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// readyz is a readiness probe.
func readyz(isReady *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if isReady == nil || !isReady.Load().(bool) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
