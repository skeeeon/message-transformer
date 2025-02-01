package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"message-transformer/internal/config"
	"message-transformer/internal/mqtt"
	"message-transformer/internal/transformer"
)

// Server represents the HTTP server
type Server struct {
	router      *chi.Mux
	logger      *zap.Logger
	rules       []config.Rule
	transformer *transformer.Transformer
	mqtt        *mqtt.Client
}

// NewServer creates a new HTTP server instance
func NewServer(
	logger *zap.Logger,
	rules []config.Rule,
	transformer *transformer.Transformer,
	mqtt *mqtt.Client,
) *Server {
	s := &Server{
		router:      chi.NewRouter(),
		logger:      logger,
		rules:       rules,
		transformer: transformer,
		mqtt:        mqtt,
	}

	s.setupMiddleware()
	s.setupRoutes()

	return s
}

// setupMiddleware configures the middleware stack
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(NewStructuredLogger(s.logger))
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(30 * time.Second))
	s.router.Use(middleware.AllowContentType("application/json"))
}

// setupRoutes configures the route handlers
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.Get("/health", s.handleHealth())

	// Dynamic rule-based endpoints
	for _, rule := range s.rules {
		s.router.Method(
			rule.API.Method,
			rule.API.Path,
			s.handleTransform(rule),
		)
	}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
