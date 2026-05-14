// File: server/ws/server_test.go
// Purpose: Verifies room management, sorting, and delete behavior at the HTTP boundary.
//
//nolint:noctx,revive // Test request helpers intentionally use concise constructors and fixture signatures.
package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	coreconfig "github.com/axwell0/elmakina/engine/core/config"
	httpapi "github.com/axwell0/elmakina/internal/api"
	sessionapp "github.com/axwell0/elmakina/internal/app/session"
	"github.com/stretchr/testify/require"
)

// The HTTP-level server tests cover room lifecycle, resumable sessions, and authorization.
func TestServerCreateRoomGeneratesUniqueIDs(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	body, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-1",
	})
	require.NoError(t, err)

	first := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, first)
	require.Equal(t, http.StatusCreated, firstRecorder.Code)
	var firstResponse httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(firstRecorder.Body.Bytes(), &firstResponse))

	second := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, second)
	require.Equal(t, http.StatusCreated, secondRecorder.Code)
	var secondResponse httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(secondRecorder.Body.Bytes(), &secondResponse))
	require.NotEqual(t, firstResponse.RoomID, secondResponse.RoomID)
}

func TestServerListRoomsReturnsSortedRooms(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	createRoom := func(deviceID string) string {
		t.Helper()
		body, err := json.Marshal(httpapi.CreateRoomRequest{
			LeaderName:  "A",
			PlayerCount: 2,
			DeviceID:    deviceID,
		})
		require.NoError(t, err)

		request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusCreated, recorder.Code)

		var created httpapi.CreateRoomResponse
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &created))
		return created.RoomID
	}

	roomA := createRoom("device-a")
	roomB := createRoom("device-b")
	expected := []string{roomA, roomB}
	sort.Strings(expected)

	request := httptest.NewRequest(http.MethodGet, "/api/rooms", http.NoBody)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code)

	var response httpapi.ListRoomsResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Rooms, 2)
	require.Equal(t, expected[0], response.Rooms[0].RoomID)
	require.Equal(t, expected[1], response.Rooms[1].RoomID)
}

func TestServerCreateRoomStoresDeviceID(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	body, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-1",
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)

	var response httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	room, ok := server.getRoom(response.RoomID)
	require.True(t, ok)
	require.Equal(t, "device-1", room.lobby.Session(response.SessionID).DeviceID)
}

func TestServerCreateRoomStoresCustomGameConfig(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)
	cfg := coreconfig.DefaultGameConfig()
	cfg.Economy.MaxCoins = 6

	body, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-1",
		GameConfig:  &cfg,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)

	var response httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	room, ok := server.getRoom(response.RoomID)
	require.True(t, ok)
	snapshot := room.lobby.Snapshot()
	require.Equal(t, 6, snapshot.GameConfig.Economy.MaxCoins)
	require.Equal(t, 6, snapshot.Game.MaxCoins)
}

func TestServerCreateRoomPersistsRoomAndLeaderSession(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	body, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-1",
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusCreated, recorder.Code)

	var response httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	roomSnapshot, err := server.stores.Rooms.LoadRoom(context.Background(), response.RoomID)
	require.NoError(t, err)
	require.Equal(t, response.RoomID, roomSnapshot.ID)
	require.Len(t, roomSnapshot.Sessions, 1)

	sessionSnapshot, err := server.stores.Sessions.LoadSession(context.Background(), response.RoomID, response.SessionID)
	require.NoError(t, err)
	require.Equal(t, "device-1", sessionSnapshot.DeviceID)
	require.Equal(t, response.PlayerIndex, sessionSnapshot.PlayerIndex)
}

func TestServerJoinRoomStoresDeviceID(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	createBody, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-leader",
	})
	require.NoError(t, err)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(createBody))
	createRecorder := httptest.NewRecorder()
	handler.ServeHTTP(createRecorder, createRequest)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var created httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(createRecorder.Body.Bytes(), &created))

	joinBody, err := json.Marshal(httpapi.JoinRoomRequest{
		PlayerName: "B",
		DeviceID:   "device-guest",
	})
	require.NoError(t, err)

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+created.RoomID+"/join", bytes.NewReader(joinBody))
	joinRecorder := httptest.NewRecorder()
	handler.ServeHTTP(joinRecorder, joinRequest)
	require.Equal(t, http.StatusOK, joinRecorder.Code)

	var joined httpapi.JoinRoomResponse
	require.NoError(t, json.Unmarshal(joinRecorder.Body.Bytes(), &joined))
	room, ok := server.getRoom(created.RoomID)
	require.True(t, ok)
	require.Equal(t, "device-guest", room.lobby.Session(joined.SessionID).DeviceID)
}

func TestServerJoinRoomPersistsRoomAndSession(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	createBody, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-leader",
	})
	require.NoError(t, err)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(createBody))
	createRecorder := httptest.NewRecorder()
	handler.ServeHTTP(createRecorder, createRequest)
	require.Equal(t, http.StatusCreated, createRecorder.Code)

	var created httpapi.CreateRoomResponse
	require.NoError(t, json.Unmarshal(createRecorder.Body.Bytes(), &created))

	joinBody, err := json.Marshal(httpapi.JoinRoomRequest{
		PlayerName: "B",
		DeviceID:   "device-guest",
	})
	require.NoError(t, err)

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/rooms/"+created.RoomID+"/join", bytes.NewReader(joinBody))
	joinRecorder := httptest.NewRecorder()
	handler.ServeHTTP(joinRecorder, joinRequest)
	require.Equal(t, http.StatusOK, joinRecorder.Code)

	var joined httpapi.JoinRoomResponse
	require.NoError(t, json.Unmarshal(joinRecorder.Body.Bytes(), &joined))
	roomSnapshot, err := server.stores.Rooms.LoadRoom(context.Background(), created.RoomID)
	require.NoError(t, err)
	require.Len(t, roomSnapshot.Sessions, 2)

	sessionSnapshot, err := server.stores.Sessions.LoadSession(context.Background(), created.RoomID, joined.SessionID)
	require.NoError(t, err)
	require.Equal(t, "device-guest", sessionSnapshot.DeviceID)
	require.Equal(t, joined.PlayerIndex, sessionSnapshot.PlayerIndex)
}

func TestServerCreateRoomRejectsMissingDeviceID(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	body, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestServerCreateRoomReturnsRoomCodeGenerationFailureImmediately(t *testing.T) {
	server := NewServer()
	handler := testHandler(server)

	originalRandomRoomCode := randomRoomCodeFunc
	randomRoomCodeFunc = func() (string, error) {
		return "", errors.New("rng blew up")
	}
	defer func() {
		randomRoomCodeFunc = originalRandomRoomCode
	}()

	body, err := json.Marshal(httpapi.CreateRoomRequest{
		LeaderName:  "A",
		PlayerCount: 2,
		DeviceID:    "device-1",
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/rooms", bytes.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var response httpapi.ErrorResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, "rng blew up", response.Error)
	require.Empty(t, server.rooms)
}

func TestServerDeleteRoomRemovesHostedSessionAndClosesClients(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	leaderSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	leaderSession.SetResumeToken("leader-secret")
	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	client := &fakeRoomClient{session: leaderSession}
	require.NoError(t, hosted.Register(client))

	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/room-1?sessionId="+leaderSession.ClientSessionID, http.NoBody)
	request.Header.Set("Authorization", "Bearer leader-secret")
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	_, exists := server.rooms["room-1"]
	require.False(t, exists)
	require.True(t, client.closed)
	require.False(t, hosted.HasClient(leaderSession.ClientSessionID))
}

func TestServerDeleteRoomRejectsInvalidResumeToken(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	leaderSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	leaderSession.SetResumeToken("leader-secret")
	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/room-1?sessionId="+leaderSession.ClientSessionID, http.NoBody)
	request.Header.Set("Authorization", "Bearer wrong-secret")
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	_, exists := server.rooms["room-1"]
	require.True(t, exists)
}

func TestServerListResumableGamesReturnsMatchingDeviceSessions(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	player, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	player.DeviceID = "device-1"
	player.SetResumeToken("secret")

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodGet, "/api/me/resumable-games?deviceId=device-1", http.NoBody)
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response httpapi.ListResumableGamesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Games, 1)
	require.Equal(t, "room-1", response.Games[0].RoomID)
	require.Equal(t, "A", response.Games[0].PlayerName)
	require.False(t, response.Games[0].Connected)
	require.True(t, response.Games[0].CanReconnect)
}

func TestServerListResumableGamesReportsConnectedFromRoomRegistry(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	player, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	player.DeviceID = "device-1"
	player.SetResumeToken("secret")

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	client := &fakeRoomClient{session: player}
	require.NoError(t, hosted.Register(client))
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodGet, "/api/me/resumable-games?deviceId=device-1", http.NoBody)
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response httpapi.ListResumableGamesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Games, 1)
	require.True(t, response.Games[0].Connected)
}

func TestServerListResumableGamesIgnoresOtherDevices(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	player, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	player.DeviceID = "device-1"

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodGet, "/api/me/resumable-games?deviceId=device-2", http.NoBody)
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response httpapi.ListResumableGamesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Empty(t, response.Games)
}

func TestServerDeleteRoomRejectsMatchingDeviceIDWithoutValidResumeToken(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewLobby("room-1", "A", 2)
	require.NoError(t, err)
	leaderSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	leaderSession.DeviceID = "device-1"
	leaderSession.SetResumeToken("leader-secret")

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/room-1?sessionId="+leaderSession.ClientSessionID, http.NoBody)
	request.Header.Set("Authorization", "Bearer wrong-secret")
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

type fakeRoomClient struct {
	session *sessionapp.ClientSession
	closed  bool
}

// Session exposes the backing client session to the room registry.
func (f *fakeRoomClient) Session() *sessionapp.ClientSession {
	return f.session
}

// Send is a no-op because these tests only care that the room can close the client later.
func (f *fakeRoomClient) Send(_ context.Context, message any) error {
	return nil
}

// Close records that the room asked the websocket to shut down.
func (f *fakeRoomClient) Close() error {
	f.closed = true
	return nil
}

// testHandler wires the server handlers into a small mux that mimics the app routes.
func testHandler(server *Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", server.HandleIndex)
	mux.HandleFunc("GET /play", server.HandleIndex)
	mux.HandleFunc("GET /api/rooms", server.HandleListRooms)
	mux.HandleFunc("POST /api/rooms", server.HandleCreateRoom)
	mux.HandleFunc("GET /api/rooms/{roomID}", server.HandleGetRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/join", server.HandleJoinRoom)
	mux.HandleFunc("POST /api/rooms/{roomID}/restart", server.HandleRestartRoom)
	mux.HandleFunc("DELETE /api/rooms/{roomID}", server.HandleDeleteRoom)
	mux.HandleFunc("GET /api/me/resumable-games", server.HandleListResumableGames)
	mux.HandleFunc("GET /ws", server.HandleWebSocket)
	return mux
}
