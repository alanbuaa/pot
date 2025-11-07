package apis

// Example Usage Documentation
//
// This file demonstrates how to use the refactored APIs module
// to support multiple consensus mechanisms through the ApiServer.
//
// Basic Setup for PoT Consensus:
//
// ```go
// import (
//     "github.com/sirupsen/logrus"
//     "github.com/zzz136454872/upgradeable-consensus/consensus/pot"
//     "github.com/zzz136454872/upgradeable-consensus/internal/apis"
// )
//
// func main() {
//     // Create logger
//     logger := logrus.New()
//     log := logger.WithField("module", "api")
//
//     // Create API server configuration
//     config := &apis.Config{
//         Port: 18025,
//     }
//
//     // Create API server instance
//     apiServer := apis.NewApiServer(config, log)
//
//     // Create or get your PoT worker instance
//     potWorker := pot.NewWorker(...)
//
//     // Create adapter and register PoT service
//     potService := apis.NewPotWorkerAdapter(potWorker)
//     apiServer.RegisterPotService(potService)
//
//     // Start the API server
//     if err := apiServer.Start(); err != nil {
//         log.Fatalf("Failed to start API server: %v", err)
//     }
//
//     // Server is now running and handling PoT-specific API requests
// }
// ```
//
// Supporting Multiple Consensus Mechanisms:
//
// ```go
// func main() {
//     logger := logrus.New()
//     log := logger.WithField("module", "api")
//
//     config := &apis.Config{Port: 18025}
//     apiServer := apis.NewApiServer(config, log)
//
//     // Register PoT consensus service
//     potWorker := pot.NewWorker(...)
//     potService := apis.NewPotWorkerAdapter(potWorker)
//     apiServer.RegisterPotService(potService)
//
//     // Register PoW consensus service (when implemented)
//     // powWorker := pow.NewWorker(...)
//     // powService := NewPowWorkerAdapter(powWorker)
//     // apiServer.RegisterPowService(powService)
//
//     // All registered consensus services are available through /api endpoints
//     apiServer.Start()
// }
// ```
//
// API Endpoints (PoT):
//
// POST /api/createlocktransaction      - Create a lock transaction
// POST /api/locktransfertransaction    - Create a lock transfer transaction
// POST /api/nonlocktransfertransaction - Create a non-lock transfer transaction
// POST /api/devastatetransaction       - Create a devastate transaction
// GET  /api/getblockheight             - Get current block height
// POST /api/hello                      - Health check endpoint
//
// Architecture Benefits:
//
// 1. Separation of Concerns:
//    - model/: Data structures and interfaces
//    - handlers/: HTTP request handling logic
//    - apis.go: Server orchestration and routing
//
// 2. Extensibility:
//    - Easy to add new consensus mechanisms by implementing the service interface
//    - New consensus types can be added without modifying existing code
//
// 3. Testability:
//    - Handlers can be tested independently with mock services
//    - Service interfaces make unit testing straightforward
//
// 4. Type Safety:
//    - Strong typing through Go interfaces
//    - Compile-time checking for interface compliance
//
// Custom Consensus Integration:
//
// To add a new consensus mechanism:
//
// 1. Define the service interface in model/your_consensus_model.go:
//    ```go
//    type YourConsensusService interface {
//        ValidateTransaction(tx *types.Tx) error
//        BroadcastTransaction(tx *types.Tx) error
//        GetCurrentHeight() uint64
//    }
//    ```
//
// 2. Implement the handler in handlers/your_consensus_handler.go:
//    ```go
//    type YourConsensusHandler struct {
//        service model.YourConsensusService
//        log     *logrus.Entry
//    }
//
//    func (h *YourConsensusHandler) RegisterRoutes(group *gin.RouterGroup) {
//        group.POST("/yourendpoint", h.HandleYourEndpoint)
//    }
//    ```
//
// 3. Add registration method to ApiServer:
//    ```go
//    func (s *ApiServer) RegisterYourConsensus(service model.YourConsensusService) {
//        handler := handlers.NewYourConsensusHandler(service, s.log)
//        s.handlers = append(s.handlers, handler)
//    }
//    ```
//
// 4. Create an adapter if needed:
//    ```go
//    type YourWorkerAdapter struct {
//        worker *yourpkg.Worker
//    }
//
//    func (a *YourWorkerAdapter) ValidateTransaction(tx *types.Tx) error {
//        return a.worker.Validate(tx)
//    }
//    ```
