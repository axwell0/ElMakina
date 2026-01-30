package replay

import (
	"time"

	"gorm.io/datatypes"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

type Match struct {
	ID              string    `gorm:"primaryKey;size:64"`
	LobbyID         string    `gorm:"size:64"`
	RulesetVersion  string    `gorm:"size:32"`
	ProtocolVersion string    `gorm:"size:32"`
	RNGSeed         string    `gorm:"size:128"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	EndedAt         *time.Time
}

type MatchParticipant struct {
	ID          uint      `gorm:"primaryKey"`
	MatchID     string    `gorm:"index;size:64"`
	PlayerID    string    `gorm:"size:64"`
	PlayerIndex int       `gorm:"index"`
	Nick        string    `gorm:"size:128"`
	Avatar      string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

type MatchEvent struct {
	ID              uint           `gorm:"primaryKey"`
	MatchID         string         `gorm:"index;size:64"`
	Seq             int64          `gorm:"index"`
	Type            string         `gorm:"size:64"`
	Visibility      string         `gorm:"size:16"`
	PlayerID        *string        `gorm:"size:64"`
	Payload         datatypes.JSON `gorm:"type:jsonb"`
	RulesetVersion  string         `gorm:"size:32"`
	ProtocolVersion string         `gorm:"size:32"`
	CreatedAt       time.Time      `gorm:"autoCreateTime"`
}

type MatchSnapshot struct {
	ID        uint           `gorm:"primaryKey"`
	MatchID   string         `gorm:"index;size:64"`
	Seq       int64          `gorm:"index"`
	Payload   datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
}

// ReplayPayload bundles the response for a replay request.
type ReplayPayload struct {
	Match          Match              `json:"match"`
	Participants   []MatchParticipant `json:"participants"`
	Events         []MatchEvent       `json:"events"`
	Snapshots      []MatchSnapshot    `json:"snapshots"`
	ViewerPlayerID string             `json:"viewer_player_id"`
	ViewerIndex    int                `json:"viewer_index"`
}
