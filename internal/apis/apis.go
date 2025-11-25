package apis

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/handlers"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis/model"
)

// Config holds the API server configuration
type Config struct {
	Port int
}

// ConsensusHandler defines the interface for consensus-specific HTTP handlers
type ConsensusHandler interface {
	RegisterRoutes(group *gin.RouterGroup)
}

// ApiServer represents the HTTP API server supporting multiple consensus mechanisms
type ApiServer struct {
	Engine          *gin.Engine
	config          *Config
	handlers        []ConsensusHandler
	monitorHandlers []ConsensusHandler
	log             *logrus.Entry
}

// NewApiServer creates a new API server instance
func NewApiServer(config *Config, log *logrus.Entry) *ApiServer {
	return &ApiServer{
		config:          config,
		handlers:        make([]ConsensusHandler, 0),
		monitorHandlers: make([]ConsensusHandler, 0),
		log:             log,
	}
}

// RegisterPotService registers PoT consensus service to the API server
func (s *ApiServer) RegisterPotService(service model.PotConsensusService) {
	handler := handlers.NewPotHandler(service, s.log)
	s.handlers = append(s.handlers, handler)
}

// RegisterPowService registers PoW consensus service to the API server
func (s *ApiServer) RegisterPowService(service model.PowConsensusService) {
	handler := handlers.NewPowHandler(service, s.log)
	s.handlers = append(s.handlers, handler)
}

// RegisterMonitorService registers monitoring service to the API server
func (s *ApiServer) RegisterMonitorService(service model.MonitorService) {
	handler := handlers.NewMonitorHandler(service, s.log)
	s.monitorHandlers = append(s.monitorHandlers, handler)
}

// setupRoutes configures all HTTP routes for registered consensus handlers
func (s *ApiServer) setupRoutes() *gin.Engine {
	r := gin.Default()

	// Enable CORS for web frontend
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Create API group
	apiGroup := r.Group("/api")

	// Register all consensus handler routes
	for _, handler := range s.handlers {
		handler.RegisterRoutes(apiGroup)
	}

	// Register all monitor handler routes
	for _, handler := range s.monitorHandlers {
		handler.RegisterRoutes(apiGroup)
	}

	return r
}

// Start starts the HTTP API server
func (s *ApiServer) Start() error {
	s.Engine = s.setupRoutes()
	addr := fmt.Sprintf("0.0.0.0:%d", s.config.Port)
	s.log.Infof("Starting API server on %s", addr)

	go func() {
		if err := s.Engine.Run(addr); err != nil {
			s.log.Fatalf("Failed to start API server: %v", err)
		}
	}()

	return nil
}

// Stop stops the HTTP API server (placeholder for graceful shutdown)
func (s *ApiServer) Stop() error {
	s.log.Info("Stopping API server")
	// TODO: Implement graceful shutdown if needed
	return nil
}
