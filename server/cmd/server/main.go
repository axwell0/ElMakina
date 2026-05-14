package main

import (
	"log"
	"net/http"
	"os"
	"time"

	httpserver "github.com/axwell0/elmakina/internal/transport/http"
	"github.com/axwell0/elmakina/server/ws"
)

func main() {
	addr := os.Getenv("ELMAKINA_HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := ws.NewServer()
	handler := httpserver.NewHandler(server)
	//nolint:gosec // Local developer log line; addr may come from env by design.
	log.Printf("El Makina local server listening on %s", addr)
	//nolint:gosec // Local developer log line; addr may come from env by design.
	log.Printf("Open http://localhost%s/ in two tabs after creating a room", addr)

	serverInstance := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := serverInstance.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
