package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/internal/apis"
	storage "github.com/zzz136454872/upgradeable-consensus/internal/storage/pot"
)

// API Server Test for PoT Consensus
//
// This test demonstrates the refactored API architecture with a minimal working setup.
// It creates a standalone PoT worker without full consensus network participation,
// allowing you to test the API endpoints independently.
//
// Usage:
//   go run main.go -config <config_file> -id <node_id>
//   go run main.go -standalone (for standalone testing mode)

var (
	configPath = flag.String("config", "../../config/config_test.yaml", "Path to config file")
	nodeID     = flag.Int64("id", 0, "Node ID")
	standalone = flag.Bool("standalone", true, "Run in standalone mode without full consensus")
	port       = flag.Int("port", 18025, "API server port")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log := logger.WithField("module", "api-test")

	log.Info("=================================================")
	log.Info("  PoT Consensus API Server Test")
	log.Info("=================================================")

	var potWorker *pot.Worker
	var err error

	if *standalone {
		// Standalone mode: minimal setup for API testing
		log.Info("Starting in STANDALONE mode (API testing only)")
		potWorker, err = createStandaloneWorker(log)
		if err != nil {
			log.Fatalf("Failed to create standalone worker: %v", err)
		}
	} else {
		// Full mode: load config and create with full network
		log.Infof("Starting in FULL mode (loading config from %s)", *configPath)
		potWorker, err = createFullWorker(*configPath, *nodeID, log)
		if err != nil {
			log.Fatalf("Failed to create worker from config: %v", err)
		}
	}

	// Create API server configuration
	apiConfig := &apis.Config{
		Port: *port,
	}

	// Create API server instance
	apiServer := apis.NewApiServer(
		apiConfig,
		log.WithField("component", "api-server"),
	)

	// Create adapter to connect PoT worker to API layer
	potService := apis.NewPotWorkerAdapter(potWorker)

	// Register PoT service with the API server
	apiServer.RegisterPotService(potService)

	// Start the API server
	if err := apiServer.Start(); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}

	log.Info("=================================================")
	log.Infof("✓ API server started successfully on port %d", apiConfig.Port)
	log.Info("=================================================")
	log.Info("Available API endpoints:")
	log.Info("  POST /api/createlocktransaction")
	log.Info("  POST /api/locktransfertransaction")
	log.Info("  POST /api/nonlocktransfertransaction")
	log.Info("  POST /api/devastatetransaction")
	log.Info("  GET  /api/getblockheight")
	log.Info("  POST /api/hello")
	log.Info("=================================================")
	log.Info("")
	log.Info("Test the API with curl:")
	log.Infof("  curl http://localhost:%d/api/hello -X POST", *port)
	log.Infof("  curl http://localhost:%d/api/getblockheight", *port)
	log.Info("")
	log.Info("Press Ctrl+C to stop the server")
	log.Info("=================================================")

	if !*standalone {
		// In full mode, start the consensus worker
		go func() {
			log.Info("Starting PoT consensus worker...")
			potWorker.Work()
		}()
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown
	if err := apiServer.Stop(); err != nil {
		log.Errorf("Error stopping API server: %v", err)
	}

	log.Info("Server stopped")
}

// createStandaloneWorker creates a minimal PoT worker for API testing
// without full consensus network setup
func createStandaloneWorker(log *logrus.Entry) (*pot.Worker, error) {
	log.Info("Creating standalone worker configuration...")

	// Create minimal consensus config
	consensusConfig := &config.ConsensusConfig{
		Type:        "pot",
		ConsensusID: 0,
		PoT: &config.PoTConfig{
			Vdf0Iteration:   100, // Minimal iterations for testing
			Vdf1Iteration:   100,
			VdfType:         "pietrzak",
			Batchsize:       10,
			ExecutorAddress: "localhost:50051", // Mock executor address
			BciRpcAddress:   ":50052",
		},
		Nodes: map[int64]*config.NodeInfo{
			0: {
				ID:         0,
				RpcAddress: "localhost:9000",
				DataDir:    "./data/node-0",
			},
		},
		Fault: 0,
		Topic: "pot-test",
	}

	// Create data directory
	dataDir := "./data/test-node"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create block storage
	blockStorage := storage.NewBlockStorage(0, dataDir)

	// Create a mock PoT engine (we'll create worker directly)
	// Note: In standalone mode, we create worker without engine to avoid network dependencies
	log.Info("Creating PoT worker...")

	potWorker := pot.NewWorker(
		0, // node ID
		consensusConfig,
		log.WithField("component", "pot-worker"),
		blockStorage,
		nil, // engine is set to nil in standalone mode
	)

	log.Info("✓ Standalone worker created successfully")
	return potWorker, nil
}

// createFullWorker creates a PoT worker with full configuration from file
func createFullWorker(configPath string, nodeID int64, log *logrus.Entry) (*pot.Worker, error) {
	log.Infof("Loading configuration from: %s", configPath)

	// Load configuration
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	log.Infof("Node ID: %d", nodeID)
	log.Infof("Total nodes: %d", cfg.Total)

	nodeInfo, err := cfg.GetNodeFromSet(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %v", err)
	}
	log.Infof("Data directory: %s", nodeInfo.DataDir)

	// Create data directory
	if err := os.MkdirAll(nodeInfo.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create block storage
	blockStorage := storage.NewBlockStorage(nodeID, nodeInfo.DataDir)

	// Create PoT worker without full engine
	// Note: This creates worker without network/executor dependencies for API testing
	potWorker := pot.NewWorker(
		nodeID,
		cfg.Consensus,
		log.WithField("component", "pot-worker"),
		blockStorage,
		nil, // engine is set to nil for API-only testing
	)

	log.Info("✓ Full worker created successfully (API-only mode)")
	log.Warn("Note: Worker created without P2P/Executor for API testing")
	return potWorker, nil
}

// Example: Supporting multiple consensus mechanisms
//
// func mainMultiConsensus() {
//     logger := logrus.New()
//     log := logger.WithField("module", "main")
//
//     // Create API server
//     apiConfig := &apis.Config{Port: 18025}
//     apiServer := apis.NewApiServer(apiConfig, log)
//
//     // Register PoT consensus
//     potWorker := createPotWorker()
//     potService := apis.NewPotWorkerAdapter(potWorker)
//     apiServer.RegisterPotService(potService)
//
//     // Register PoW consensus (when implemented)
//     // powWorker := createPowWorker()
//     // powService := apis.NewPowWorkerAdapter(powWorker)
//     // apiServer.RegisterPowService(powService)
//
//     // Start API server - now serves both consensus mechanisms
//     apiServer.Start()
//
//     // Run consensus workers
//     go potWorker.Work()
//     // go powWorker.Work()
//
//     // Wait for shutdown signal
//     waitForShutdown()
// }
