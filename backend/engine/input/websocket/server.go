package websocket

import (
	"net/http"
)

// Server wraps a Provider with an HTTP server and mux for manual testing or local play.
//
// Intent:
//
//	Keep transport wiring out of engine code while offering a ready-to-use HTTP entrypoint.
//	This is intentionally minimal; production deployments can wrap Provider with their
//	own routing, auth, and observability.
//
// Behavior:
//   - Registers the WS provider at /ws.
//   - Exposes Handler for embedding in larger servers.
//   - Does not start listening unless ListenAndServe is called.
type Server struct {
	Provider *Provider
	mux      *http.ServeMux
	server   *http.Server
}

// NewServer constructs a wrapper server with /ws mapped to the provider.
func NewServer(addr string, provider *Provider) *Server {
	if provider == nil {
		provider = NewProvider(nil)
	}
	mux := http.NewServeMux()
	mux.Handle("/ws", provider)
	return &Server{
		Provider: provider,
		mux:      mux,
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// Handler returns the underlying mux for integration with other HTTP routes.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe starts the HTTP server. Use Close to stop it.
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Close stops the HTTP server.
func (s *Server) Close() error {
	return s.server.Close()
}
