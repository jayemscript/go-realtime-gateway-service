package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jayemscript/go-realtime-gateway-service/internal/broker"
	"github.com/jayemscript/go-realtime-gateway-service/internal/config"
)

func TestHealthReturnsOKAndRequestID(t *testing.T) {
	server := newTestServer()
	recorder := serveRequest(server, http.MethodGet, "/health")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if !strings.HasPrefix(recorder.Header().Get("X-Request-ID"), "req_") {
		t.Fatalf("X-Request-ID = %q, want req_ prefix", recorder.Header().Get("X-Request-ID"))
	}

	var body map[string]string
	decodeJSON(t, recorder, &body)
	if body["status"] != "ok" {
		t.Fatalf("status body = %q, want ok", body["status"])
	}
}

func TestReadyTracksLifecycle(t *testing.T) {
	server := newTestServer()

	initial := serveRequest(server, http.MethodGet, "/ready")
	if initial.Code != http.StatusServiceUnavailable {
		t.Fatalf("initial status = %d, want %d", initial.Code, http.StatusServiceUnavailable)
	}

	server.ready.Store(true)
	ready := serveRequest(server, http.MethodGet, "/ready")
	if ready.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d", ready.Code, http.StatusOK)
	}

	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	afterShutdown := serveRequest(server, http.MethodGet, "/ready")
	if afterShutdown.Code != http.StatusServiceUnavailable {
		t.Fatalf("after shutdown status = %d, want %d", afterShutdown.Code, http.StatusServiceUnavailable)
	}
}

func TestHealthRejectsUnsupportedMethod(t *testing.T) {
	server := newTestServer()
	recorder := serveRequest(server, http.MethodPost, "/health")

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", got)
	}

	var body errorResponse
	decodeJSON(t, recorder, &body)
	if body.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Fatalf("error code = %q, want METHOD_NOT_ALLOWED", body.Error.Code)
	}
	if body.Error.RequestID == "" {
		t.Fatal("error response requestId is empty")
	}
}

func TestUnknownRouteReturnsJSONNotFound(t *testing.T) {
	server := newTestServer()
	recorder := serveRequest(server, http.MethodGet, "/missing")

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}

	var body errorResponse
	decodeJSON(t, recorder, &body)
	if body.Error.Code != "NOT_FOUND" {
		t.Fatalf("error code = %q, want NOT_FOUND", body.Error.Code)
	}
}

func newTestServer() *Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(config.Config{HTTPAddr: "127.0.0.1:0"}, logger, broker.NewMemory())
}

func serveRequest(server *Server, method, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	server.httpServer.Handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeJSON(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.NewDecoder(recorder.Body).Decode(target); err != nil {
		t.Fatalf("decode response JSON: %v", err)
	}
}
