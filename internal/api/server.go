package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jayemscript/go-realtime-gateway-service/internal/config"
)

// Server owns the Phase 1 HTTP lifecycle. Routes are added in later phases.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func NewServer(cfg config.Config, logger *slog.Logger) *Server {
	mux := http.NewServeMux()

	return &Server{
		logger: logger,
		httpServer: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
	}
}

func (s *Server) Start() error {
	s.logger.Info("gateway starting", "address", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("gateway shutting down")
	return s.httpServer.Shutdown(ctx)
}
