package ws

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/axwell0/elmakina/internal/apperrors"
)

func roomNotFoundError(roomID string) error {
	return fmt.Errorf("room %q: %w", roomID, apperrors.ErrRoomNotFound)
}

func sessionNotFoundError(roomID, sessionID string) error {
	return fmt.Errorf("session %q in room %q: %w", sessionID, roomID, apperrors.ErrSessionNotFound)
}

func unauthorizedError(reason string) error {
	if reason == "" {
		reason = apperrors.ErrUnauthorizedCommand.Error()
	}
	return fmt.Errorf("%s: %w", reason, apperrors.ErrUnauthorizedCommand)
}

func httpStatusForError(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, apperrors.ErrRoomNotFound), errors.Is(err, apperrors.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperrors.ErrUnauthorizedCommand):
		return http.StatusForbidden
	case errors.Is(err, apperrors.ErrInvalidPhase), errors.Is(err, apperrors.ErrNoPrompt), errors.Is(err, apperrors.ErrStalePrompt):
		return http.StatusBadRequest
	default:
		return http.StatusBadRequest
	}
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, httpStatusForError(err), errorResponse(err))
}

func errorResponse(err error) any {
	if err == nil {
		return nil
	}
	return struct {
		Error string `json:"error"`
	}{Error: err.Error()}
}
