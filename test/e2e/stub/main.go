// Command stub is a tiny readiness-serving HTTP stand-in used only by the kind
// e2e smoke (test/e2e). It answers 200 on every path (so the operator's
// hard-coded /healthz + /readyz probes pass) and listens on both Zelos
// component ports (8000 and 8080) at once, so a single image can stand in for
// every component (gateway/mcp/server on 8000, broker/backplane on 8080)
// without pulling private GHCR images into the CI cluster.
//
// It is NOT a real Zelos component and must never be shipped — it exists so the
// e2e smoke can exercise the operator's status/readiness roll-up end-to-end on
// a cluster with no registry credentials.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "ok %s\n", r.URL.Path)
}

func serve(addr string, wg *sync.WaitGroup) {
	defer wg.Done()
	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	log.Printf("e2e stub listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("listen %s: %v", addr, err)
		os.Exit(1)
	}
}

func main() {
	var wg sync.WaitGroup
	for _, addr := range []string{":8000", ":8080"} {
		wg.Add(1)
		go serve(addr, &wg)
	}
	wg.Wait()
}
