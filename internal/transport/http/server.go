package httpserver

import (
	"net/http"

	wsserver "github.com/axwell0/elmakina/server/ws"
)

func NewHandler(server *wsserver.Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /", server.HandleIndex)                                    // index
	mux.HandleFunc("GET /play", server.HandleIndex)                                // play path
	mux.HandleFunc("GET /api/rooms", server.HandleListRooms)                       // list all the rooms (lobbies)
	mux.HandleFunc("GET /api/me/resumable-games", server.HandleListResumableGames) //
	mux.HandleFunc("POST /api/rooms", server.HandleCreateRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}", server.HandleGetRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/join", server.HandleJoinRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/restart", server.HandleRestartRoom)
	mux.HandleFunc("DELETE /api/rooms/{roomID}", server.HandleDeleteRoom)
	mux.HandleFunc("GET /ws", server.HandleWebSocket)
	return mux
}
