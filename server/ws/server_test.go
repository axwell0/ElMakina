// File: server/ws/server_test.go
// Purpose: Verifies room management, sorting, and delete behavior at the HTTP boundary.
package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	sessionapp "example.com/elmakina/app/session"
	httpapi "example.com/elmakina/server/httpapi"
	"github.com/stretchr/testify/require"
)

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

	request := httptest.NewRequest(http.MethodGet, "/api/rooms", nil)
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
	require.Equal(t, "device-1", room.session.Session(response.SessionID).DeviceID)
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
	require.Equal(t, "device-guest", room.session.Session(joined.SessionID).DeviceID)
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

func TestServerDeleteRoomRemovesHostedSessionAndClosesClients(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, nil)
	require.NoError(t, err)
	leaderSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	delete(session.Sessions, leaderSession.ID)
	leaderSession.ID = "leader-session"
	leaderSession.SetResumeToken("leader-secret")
	session.Sessions[leaderSession.ID] = leaderSession
	client := &fakeRoomClient{session: leaderSession}
	require.NoError(t, hosted.Register(client))

	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/room-1?sessionId=leader-session", nil)
	request.Header.Set("Authorization", "Bearer leader-secret")
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	_, exists := server.rooms["room-1"]
	require.False(t, exists)
	require.True(t, client.closed)
	require.False(t, leaderSession.Connected)
}

func TestServerDeleteRoomRejectsInvalidResumeToken(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, nil)
	require.NoError(t, err)
	leaderSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	leaderSession.SetResumeToken("leader-secret")
	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	delete(session.Sessions, leaderSession.ID)
	leaderSession.ID = "leader-session"
	session.Sessions[leaderSession.ID] = leaderSession
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/room-1?sessionId=leader-session", nil)
	request.Header.Set("Authorization", "Bearer wrong-secret")
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	_, exists := server.rooms["room-1"]
	require.True(t, exists)
}

func TestServerListResumableGamesReturnsMatchingDeviceSessions(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, nil)
	require.NoError(t, err)
	player, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	player.DeviceID = "device-1"
	player.SetResumeToken("secret")
	player.Disconnect(time.Unix(10, 0).UTC())

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodGet, "/api/me/resumable-games?deviceId=device-1", nil)
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response httpapi.ListResumableGamesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Games, 1)
	require.Equal(t, "room-1", response.Games[0].RoomID)
	require.Equal(t, "A", response.Games[0].PlayerName)
	require.True(t, response.Games[0].CanReconnect)
}

func TestServerListResumableGamesIgnoresOtherDevices(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, nil)
	require.NoError(t, err)
	player, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	player.DeviceID = "device-1"

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodGet, "/api/me/resumable-games?deviceId=device-2", nil)
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response httpapi.ListResumableGamesResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Empty(t, response.Games)
}

func TestServerDeleteRoomRejectsMatchingDeviceIDWithoutValidResumeToken(t *testing.T) {
	server := NewServer()

	session, err := sessionapp.NewGameSession("room-1", "A", 2, nil)
	require.NoError(t, err)
	leaderSession, err := session.JoinPlayer("A", time.Unix(0, 0).UTC())
	require.NoError(t, err)
	leaderSession.DeviceID = "device-1"
	leaderSession.SetResumeToken("leader-secret")

	hosted, err := newRoom(session, server.turnConfig, server.logger)
	require.NoError(t, err)
	server.rooms["room-1"] = hosted

	request := httptest.NewRequest(http.MethodDelete, "/api/rooms/room-1?sessionId="+leaderSession.ID, nil)
	request.Header.Set("Authorization", "Bearer wrong-secret")
	recorder := httptest.NewRecorder()
	testHandler(server).ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}

type fakeRoomClient struct {
	session *sessionapp.ClientSession
	closed  bool
}

func (f *fakeRoomClient) Session() *sessionapp.ClientSession {
	return f.session
}

func (f *fakeRoomClient) Send(_ context.Context, message any) error {
	return nil
}

func (f *fakeRoomClient) Close() error {
	f.closed = true
	return nil
}

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
