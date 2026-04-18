package main

import (
	"log"
	"net/http"
	"os"

	ws "example.com/elmakina/server/ws"
)

func main() {
	port := os.Getenv("ELMAKINA_PORT")
	if port == "" {
		port = ":8080"
	}

	server := ws.NewServer()
	log.Printf("El Makina local server listening on %s", port)
	log.Printf("Open http://localhost%s/ in two tabs after creating a room", port)

	if err := http.ListenAndServe(port, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
