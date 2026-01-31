package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRegistersWSHandler(t *testing.T) {
	s := NewServer(":0", NewProvider(nil))

	req := httptest.NewRequest(http.MethodGet, "http://example/ws", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("expected /ws handler to be registered")
	}
}

func TestServerOtherPathNotFound(t *testing.T) {
	s := NewServer(":0", NewProvider(nil))

	req := httptest.NewRequest(http.MethodGet, "http://example/other", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", rec.Code)
	}
}
