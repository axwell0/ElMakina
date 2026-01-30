package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ElMakina/backend/engine/state"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Recorder interface {
	StartMatch(ctx context.Context, input MatchStartInput) error
	RecordEvent(ctx context.Context, input EventInput) error
	Snapshot(ctx context.Context, input SnapshotInput) error
	EndMatch(ctx context.Context, input MatchEndInput) error
	LoadReplay(ctx context.Context, matchID string, viewerPlayerID string) (*ReplayPayload, error)
}

type Store struct {
	db *gorm.DB
}

type MatchStartInput struct {
	MatchID         string
	LobbyID         string
	Players         []PlayerInfo
	RulesetVersion  string
	ProtocolVersion string
	RNGSeed         string
}

type PlayerInfo struct {
	PlayerID    string
	PlayerIndex int
	Nick        string
	Avatar      string
}

type EventInput struct {
	MatchID         string
	Seq             int64
	Type            string
	Visibility      string
	PlayerID        *string
	RulesetVersion  string
	ProtocolVersion string
	Payload         any
}

type SnapshotInput struct {
	MatchID string
	Seq     int64
	State   *state.GameState
}

type MatchEndInput struct {
	MatchID         string
	WinnerIndex     int
	WinnerName      string
	EndedAt         time.Time
	RulesetVersion  string
	ProtocolVersion string
}

func OpenPostgres(ctx context.Context, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

func (s *Store) AutoMigrate(ctx context.Context) error {
	return s.db.WithContext(ctx).AutoMigrate(&Match{}, &MatchParticipant{}, &MatchEvent{}, &MatchSnapshot{})
}

func (s *Store) StartMatch(ctx context.Context, input MatchStartInput) error {
	if input.MatchID == "" {
		return fmt.Errorf("match_id is required")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		match := Match{
			ID:              input.MatchID,
			LobbyID:         input.LobbyID,
			RulesetVersion:  input.RulesetVersion,
			ProtocolVersion: input.ProtocolVersion,
			RNGSeed:         input.RNGSeed,
			CreatedAt:       time.Now(),
		}
		if err := tx.Create(&match).Error; err != nil {
			return err
		}
		for _, player := range input.Players {
			participant := MatchParticipant{
				MatchID:     input.MatchID,
				PlayerID:    player.PlayerID,
				PlayerIndex: player.PlayerIndex,
				Nick:        player.Nick,
				Avatar:      player.Avatar,
				CreatedAt:   time.Now(),
			}
			if err := tx.Create(&participant).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) RecordEvent(ctx context.Context, input EventInput) error {
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return err
	}
	event := MatchEvent{
		MatchID:         input.MatchID,
		Seq:             input.Seq,
		Type:            input.Type,
		Visibility:      input.Visibility,
		PlayerID:        input.PlayerID,
		Payload:         datatypes.JSON(payload),
		RulesetVersion:  input.RulesetVersion,
		ProtocolVersion: input.ProtocolVersion,
		CreatedAt:       time.Now(),
	}
	return s.db.WithContext(ctx).Session(&gorm.Session{SkipDefaultTransaction: true}).Create(&event).Error
}

func (s *Store) recordEventsBatch(ctx context.Context, events []MatchEvent) error {
	if len(events) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Session(&gorm.Session{SkipDefaultTransaction: true}).Create(&events).Error
}

func (s *Store) Snapshot(ctx context.Context, input SnapshotInput) error {
	if input.State == nil {
		return fmt.Errorf("state is required")
	}
	payload, err := json.Marshal(input.State)
	if err != nil {
		return err
	}
	snapshot := MatchSnapshot{
		MatchID:   input.MatchID,
		Seq:       input.Seq,
		Payload:   datatypes.JSON(payload),
		CreatedAt: time.Now(),
	}
	return s.db.WithContext(ctx).Session(&gorm.Session{SkipDefaultTransaction: true}).Create(&snapshot).Error
}

func (s *Store) recordSnapshotsBatch(ctx context.Context, snapshots []MatchSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Session(&gorm.Session{SkipDefaultTransaction: true}).Create(&snapshots).Error
}

func (s *Store) EndMatch(ctx context.Context, input MatchEndInput) error {
	return s.db.WithContext(ctx).Model(&Match{}).Where("id = ?", input.MatchID).Updates(map[string]any{
		"ended_at":         input.EndedAt,
		"ruleset_version":  input.RulesetVersion,
		"protocol_version": input.ProtocolVersion,
	}).Error
}

func (s *Store) LoadReplay(ctx context.Context, matchID string, viewerPlayerID string) (*ReplayPayload, error) {
	if matchID == "" {
		return nil, fmt.Errorf("match_id is required")
	}
	var match Match
	if err := s.db.WithContext(ctx).First(&match, "id = ?", matchID).Error; err != nil {
		return nil, err
	}
	var participants []MatchParticipant
	if err := s.db.WithContext(ctx).Where("match_id = ?", matchID).Order("player_index").Find(&participants).Error; err != nil {
		return nil, err
	}

	viewerIndex := -1
	for _, p := range participants {
		if viewerPlayerID != "" && p.PlayerID == viewerPlayerID {
			viewerIndex = p.PlayerIndex
			break
		}
	}

	query := s.db.WithContext(ctx).Where("match_id = ?", matchID)
	if viewerPlayerID != "" {
		query = query.Where("visibility = ? OR (visibility = ? AND player_id = ?)", VisibilityPublic, VisibilityPrivate, viewerPlayerID)
	} else {
		query = query.Where("visibility = ?", VisibilityPublic)
	}
	var events []MatchEvent
	if err := query.Order("seq").Find(&events).Error; err != nil {
		return nil, err
	}

	var snapshots []MatchSnapshot
	if err := s.db.WithContext(ctx).Where("match_id = ?", matchID).Order("seq").Find(&snapshots).Error; err != nil {
		return nil, err
	}
	masked := make([]MatchSnapshot, 0, len(snapshots))
	for _, snap := range snapshots {
		maskedSnap, err := maskSnapshot(snap, viewerIndex)
		if err != nil {
			return nil, err
		}
		masked = append(masked, maskedSnap)
	}

	return &ReplayPayload{
		Match:          match,
		Participants:   participants,
		Events:         events,
		Snapshots:      masked,
		ViewerPlayerID: viewerPlayerID,
		ViewerIndex:    viewerIndex,
	}, nil
}

func maskSnapshot(snapshot MatchSnapshot, viewerIndex int) (MatchSnapshot, error) {
	var gs state.GameState
	if err := json.Unmarshal(snapshot.Payload, &gs); err != nil {
		return snapshot, err
	}
	masked := maskGameState(&gs, viewerIndex)
	payload, err := json.Marshal(masked)
	if err != nil {
		return snapshot, err
	}
	snapshot.Payload = datatypes.JSON(payload)
	return snapshot, nil
}

func maskGameState(gs *state.GameState, viewerIndex int) *state.GameState {
	if gs == nil {
		return nil
	}
	masked := *gs
	masked.Players = make([]state.Player, len(gs.Players))
	for i, player := range gs.Players {
		maskedPlayer := player
		if viewerIndex < 0 || i != viewerIndex {
			maskedPlayer.Hand = make([]state.Card, len(player.Hand))
		}
		masked.Players[i] = maskedPlayer
	}
	masked.Deck = make(state.Deck, len(gs.Deck))
	return &masked
}
