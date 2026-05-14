//nolint:err113,wrapcheck,gocritic // Helper utilities prioritize compact HTTP/JSON plumbing behavior.
package ws

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	api "github.com/axwell0/elmakina/internal/api"
)

// decodeJSON reads the request body once, rejects unknown fields, and insists
// that the payload contain exactly one JSON value.
func decodeJSON(body io.ReadCloser, dest any) error {
	defer body.Close()

	// Use a strict decoder so the API rejects unexpected fields instead of silently ignoring them.
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dest); err != nil {
		return err
	}

	// Consume any trailing token so the body contains exactly one JSON document.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("request body must contain only one JSON value")
	}

	return nil
}

// writeJSON emits a JSON response body with the requested status code.
func writeJSON(w http.ResponseWriter, status int, value any) {
	// Set the content type before the status so clients know to expect JSON.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}

// incomingEnvelope keeps the original websocket payload so the handler can
// switch on type first and decode the concrete message later.
type incomingEnvelope struct {
	Type api.MessageType `json:"type"`
	Raw  json.RawMessage
}

// UnmarshalJSON preserves the original payload so we can decode it later after
// switching on the message type.
func (e *incomingEnvelope) UnmarshalJSON(data []byte) error {
	// Decode only the message type first so the caller can switch on it safely.
	var decoded struct {
		Type api.MessageType `json:"type"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	e.Type = decoded.Type
	e.Raw = append(e.Raw[:0], data...)
	return nil
}

// randomRoomCode generates a short, human-readable room code from a safe alphabet.
func randomRoomCode() (string, error) {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6

	// Fill the code with random alphabet characters while rejecting out-of-range bytes.
	code := make([]byte, length)
	random := make([]byte, length)

	for filled := 0; filled < length; {
		if _, err := cryptorand.Read(random); err != nil {
			return "", err
		}

		for _, b := range random {
			if b >= 252 {
				continue
			}
			// Map the byte into the room-code alphabet and advance the output cursor.
			code[filled] = alphabet[b%36]
			filled++
			if filled == length {
				break
			}
		}
	}

	return string(code), nil
}

// generateResumeToken creates a URL-safe bearer token for reconnecting a session.
func generateResumeToken() (string, error) {
	// Tokens are raw random bytes encoded in URL-safe base64.
	// A 32-byte token gives enough entropy for session recovery.
	buf := make([]byte, 32)
	if _, err := cryptorand.Read(buf); err != nil {
		return "", fmt.Errorf("generate resume token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

var randomRoomCodeFunc = randomRoomCode

var generateResumeTokenFunc = generateResumeToken

// urlQueryEscape preserves spaces as `%20` so generated room links are stable.
func urlQueryEscape(value string) string {
	// Replace `+` with `%20` so spaces survive in a more literal form inside URLs.
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

// bearerToken extracts a strict `Bearer ...` token from the request headers.
func bearerToken(r *http.Request) string {
	// Keep the parser strict so arbitrary auth headers do not get mistaken for a resume token.
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	// Strip the prefix after we know the header is actually a bearer token.
	return strings.TrimPrefix(auth, prefix)
}

// websocketResumeAuth finds the resume token that the browser passes in the websocket subprotocol list.
func websocketResumeAuth(r *http.Request) (token string, subprotocol string, err error) {
	// Resume tokens ride in the websocket subprotocol list, prefixed so the server can recognize them.
	const prefix = "resume."
	// Scan the client-negotiated subprotocols until we find the resume token entry.
	for _, candidate := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		candidate = strings.TrimSpace(candidate)
		if strings.HasPrefix(candidate, prefix) {
			return strings.TrimPrefix(candidate, prefix), candidate, nil
		}
	}
	return "", "", fmt.Errorf("missing resume token")
}
