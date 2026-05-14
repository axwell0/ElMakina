package session

import (
	"crypto/sha256"
	"crypto/subtle"
	"time"
)

// ClientSession tracks a participant identity independently from transport.
type ClientSession struct {
	ClientSessionID string
	PlayerIndex     int
	PlayerName      string
	// DeviceID groups sessions by browser/device so the server can list resumable
	// games later
	DeviceID        string
	LastSeenAt      time.Time
	resumeTokenHash [32]byte
}

// NewClientSession creates a reconnectable client session identity.
func NewClientSession(id string, playerIndex int, playerName string) *ClientSession {
	return &ClientSession{
		ClientSessionID: id,
		PlayerIndex:     playerIndex,
		PlayerName:      playerName,
		LastSeenAt:      time.Now().UTC(),
	}
}

// Snapshot returns a persistence-safe view of the client session.
func (s *ClientSession) Snapshot() ClientSessionSnapshot {
	if s == nil {
		return ClientSessionSnapshot{}
	}
	return ClientSessionSnapshot{
		ClientSessionID: s.ClientSessionID,
		PlayerIndex:     s.PlayerIndex,
		PlayerName:      s.PlayerName,
		DeviceID:        s.DeviceID,
		LastSeenAt:      s.LastSeenAt,
	}
}

// MarkSeen updates the last-seen timestamp.
func (s *ClientSession) MarkSeen(now time.Time) {
	if s == nil {
		return
	}
	s.LastSeenAt = now.UTC()
}

// Disconnect records when the live transport disconnected without deleting the durable session.
func (s *ClientSession) Disconnect(now time.Time) {
	if s == nil {
		return
	}
	s.LastSeenAt = now.UTC()
}

// SetResumeToken stores a hashed reconnect secret for later verification.
func (s *ClientSession) SetResumeToken(token string) {
	if s == nil {
		return
	}
	s.resumeTokenHash = sha256.Sum256([]byte(token))
}

// VerifyResumeToken reports whether token matches the stored reconnect secret.
func (s *ClientSession) VerifyResumeToken(token string) bool {
	if s == nil || token == "" {
		return false
	}
	candidate := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(s.resumeTokenHash[:], candidate[:]) == 1
}
