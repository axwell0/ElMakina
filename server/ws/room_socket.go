package ws

import sessionapp "example.com/elmakina/app/session"

// roomSocket hosts transport-facing helpers as room ownership continues to split.
type roomSocket struct {
	room *Room
}

func newRoomSocket(room *Room) *roomSocket {
	return &roomSocket{room: room}
}

func (rs *roomSocket) register(client sessionapp.Client) error {
	return rs.room.Register(client)
}

func (rs *roomSocket) unregisterIfCurrent(sessionID string, client sessionapp.Client) bool {
	return rs.room.UnregisterIfCurrent(sessionID, client)
}
