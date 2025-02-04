//file: internal/api/router.go

package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"message-transformer/internal/config"
	"message-transformer/internal/mqtt"
	"message-transformer/internal/transformer"
)

// ServerConfig holds the configuration for the HTTP server
type ServerConfig struct {
	Logger      *zap.Logger
	Rules       []config.Rule
	Transformer *transformer.Transformer
	MQTT        *mqtt.Client
}

// Server represents the HTTP server
type Server struct {
	router      *chi.Mux
	logger      *zap.Logger
	rules       []config.Rule
	ruleMap     map[string]config.Rule
	transformer *transformer.Transformer
	mqtt        *mqtt.Client
	bufferPool  *sync.Pool
}

// NewServer creates a new HTTP server instance
func NewServer(cfg ServerConfig) *Server {
	// Initialize rule map for O(1) lookups
	ruleMap := make(map[string]config.Rule, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		ruleMap[rule.API.Path] = rule
	}

	s := &Server{
		router:      chi.NewRouter(),
		logger:      cfg.Logger,
		rules:       cfg.Rules,
		ruleMap:     ruleMap,
		transformer: cfg.Transformer,
		mqtt:        cfg.MQTT,
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, 32*1024) // 32KB initial buffer
			},
		},
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

	// Dynamic rule-based endpoints using pre-built rule map
	for path, rule := range s.ruleMap {
		// Capture rule in local variable for closure
		r := rule
		s.router.Method(r.API.Method, path, s.handleTransform(r))
		s.logger.Debug("Registered route",
			zap.String("method", r.API.Method),
			zap.String("path", path),
			zap.String("rule_id", r.ID))
	}
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// GetRule retrieves a rule by path with O(1) complexity
func (s *Server) GetRule(path string) (config.Rule, bool) {
	rule, exists := s.ruleMap[path]
	return rule, exists
}
