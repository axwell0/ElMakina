package httpserver

import (
	"net/http"

	wsserver "example.com/elmakina/server/ws"
)

func NewHandler(server *wsserver.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.HandleIndex)
	mux.HandleFunc("GET /play", server.HandleIndex)
	mux.HandleFunc("GET /api/rooms", server.HandleListRooms)
	mux.HandleFunc("POST /api/rooms", server.HandleCreateRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}", server.HandleGetRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/join", server.HandleJoinRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/restart", server.HandleRestartRoom)
	mux.HandleFunc("DELETE /api/rooms/{roomID}", server.HandleDeleteRoom)
	mux.HandleFunc("GET /ws", server.HandleWebSocket)
	return mux
}
