## Purpose

This file gives concise, repo-specific guidance for AI coding agents (Copilot/automation) so they can be productive quickly. It focuses on the architecture, build/test/run workflows, project-specific conventions, integration points and concrete examples from this codebase.

## Big-picture architecture

**POT (Proof of Time) 可升级共识框架** - 研究型区块链共识系统,核心特性是共识热切换能力。

### Component Layout
- **Multi-binary Go repository**: Primary entry points in `cmd/` (server, executor, client, genkey). Building outputs to `bin/`.
- **Consensus layer** (`consensus/`): 可插拔共识算法 - HotStuff (3 modes), POW, POT, Whirly, POS (开发中). Key orchestration in `consensus/upgrade/manager.go`.
- **升级系统** (`consensus/upgrade/`): 核心创新点 - MultiChainManager, MetricsCollector, PreexecMonitor, SwitchManager, RollbackManager 实现共识无缝切换.
- **Networking** (`p2p/`, `p2p/p2p-adaptor/`): P2P layer with local module replacement (`replace p2padaptor => ./p2p/p2p-adaptor` in `go.mod`).
- **API层** (`internal/apis/`): Gin-based HTTP server exposing upgrade controls, consensus metrics, monitoring. Handlers follow `RegisterRoutes(group *gin.RouterGroup)` pattern.
- **Storage** (`internal/storage/`): Multi-chain LevelDB/BoltDB persistence for main chain + candidate chains.
- **Protobufs** (`pkg/proto/*.proto`): Message definitions for consensus protocols, execution, networking. Generated Go lives beside `.proto` files.
- **Web Frontend** (`web/`): Vue 3 + TypeScript 可视化大屏 (3D particles, network topology, real-time metrics).

Key files: `main.go` spawns demo nodes; `cmd/server/server.go` runs consensus node; `consensus/upgrade/manager.go` orchestrates upgrades; `tests/upgrade/consensus_switch_test.go` shows end-to-end upgrade workflow.

## Build / test / run (concrete commands)

**Build**:
- `make build` - builds core binaries (genkey, client, server, test) to `bin/`
- `make build-cmds` - builds ALL cmd/* programs (includes bci, executor, governance, http, txtest, upgrade-cli)
- Generic pattern: `make build-<name>` builds `cmd/<name>` → `bin/<name>`

**Keys & Setup**:
- `make run_genkey` - generates keys to `data/keys/` (3 nodes, 4 key threshold). **Required before first run**.
- Keys are used by tests and runtime. Always regenerate after `data/keys/` deletion.

**Run Services**:
- `make run_server` - starts consensus node (reads `config/config.yaml`, listens on ports 6060/7070/8088)
- `make run_executor` - starts transaction executor (port 9877)
- `make run_client` - runs client tool
- Services communicate: server → executor (gRPC), server ← clients (HTTP API)

**Testing**:
- `make test` - runs all tests (`go test -v ./...`), clears test cache first
- Key test suites:
  - `tests/upgrade/consensus_switch_test.go` - 端到端升级流程 (提案→预执行→切换→回滚)
  - `internal/apis/handlers/upgrade_integration_test.go` - API integration tests
  - `consensus/pot/pot_test.go` - POT consensus unit tests

**Proto Codegen**:
- `make compile_proto` - regenerates Go from `.proto` files. Run BEFORE building if you modify `pkg/proto/*.proto`.

**Docker**:
- `make docker_build` - builds production image
- `make docker_build_all` - builds binaries + web frontend + Docker image
- `docker-compose up` - runs executor + server together (uses `config/config_docker.yaml`)

**Web Frontend** (Vue 3 visualization):
- `cd web && npm install && npm run dev` - dev server (Mock mode, no backend needed)
- `cd web && npm run build` - production build
- Toggle Mock/Real: edit `web/.env.development` (`VITE_USE_MOCK=true/false`)

## Project-specific conventions & patterns

**File Organization**:
- Binaries → `bin/`, keys → `data/keys/`, configs → `config/` (config.yaml, config_docker.yaml, config_prod.yaml)
- Node data → `data/node-{id}/`, network data → `data/network-{id}/`
- Protobufs: `.proto` and generated `.pb.go` colocated in `pkg/proto/`

**Go Module Patterns**:
- Module path: `github.com/zzz136454872/upgradeable-consensus`
- Local `replace` directives in `go.mod` for `p2padaptor` and `blockchain-crypto` - **do not break these paths**
- Import pattern: `import pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"`

**Consensus Interface** (`consensus/model/consensus.go`):
```go
type Consensus interface {
    ProcessProposal(*types.Block) error
    GetConsensusType() string
    GetHeight() uint64
    Broadcast(msg proto.Message) error
    // ... (check file for full interface)
}
```
All consensus implementations (POT, HotStuff, POW, Whirly) must satisfy this interface.

**Upgrade System Architecture** (7-phase workflow):
1. **Proposal Creation** - define target consensus, fork height, switch height
2. **Candidate Chain Preparation** - fork from main chain, initialize storage
3. **Preexecution** - run candidate consensus parallel to main chain
4. **Metrics Collection** - compare performance (TPS, latency, throughput)
5. **Governance Decision** - approve/reject based on metrics
6. **Consensus Switch** - atomic cutover at switch height
7. **Rollback (if needed)** - revert to main chain on failure

**API Handler Pattern** (`internal/apis/handlers/*_handler.go`):
```go
type XxxHandler struct {
    service model.XxxService
    log     *logrus.Entry
}
func (h *XxxHandler) RegisterRoutes(group *gin.RouterGroup) {
    group.GET("/endpoint", h.HandlerMethod)
}
```
Handlers are auto-registered in `internal/apis/apis.go` via `handler.RegisterRoutes(apiGroup)`.

**Config Loading**:
- YAML files parsed by `config/config.go`
- Multi-consensus support: `config.ConsensusConfig` with `HotStuff`, `Pow`, `PoT`, `Whirly` sub-configs
- Nodes defined in `ReplicaInfo` array with `Address`, `PubKey`, `Datadir`

**Logging**:
- Centralized via `pkg/logging`, uses logrus
- Pattern: `log := logrus.NewEntry(logrus.New())` then `log.Infof("[PoT]\tmessage")`
- Profiling: `pprof` hooked in `main.go` (CPU/memory profiling available)

## Integration points & external dependencies

**Protobuf Codegen**:
- Source: `pkg/proto/*.proto` (hotstuff.proto, pot.proto, executor.proto, common.proto, pow.proto, whirly.proto)
- Generated: co-located `.pb.go` files
- Flags: `--go_out=. --go-grpc_out=require_unimplemented_servers=false:.` (defined in Makefile)
- **Critical**: Always run `make compile_proto` after modifying `.proto` files, BEFORE building

**Local Module Dependencies** (via go.mod `replace`):
- `p2padaptor` → `./p2p/p2p-adaptor` - P2P networking adapter
- `blockchain-crypto` → `./crypto/blockchain-crypto` - Crypto primitives (VDF, BLS, threshold signatures)
- **Never** break these relative paths or refactor without updating `go.mod`

**Executor Communication** (gRPC):
- Server → Executor: gRPC at `localhost:9877` (default)
- Protocol: `pkg/proto/executor.proto` defines `Executor` service
- Methods: `Execute(ExecBlock) returns (Result)` for block execution
- Config: `config.ConsensusConfig.PoT.ExecutorAddress`

**Docker Environment**:
- Base image: `golang:1.22` (builder), `debian:bookworm-slim` (runtime)
- APT mirrors: hardcoded to Tsinghua (China), may need adjustment for international builds
- Multi-stage: separate `executor` and `server` targets in single Dockerfile
- Compose: orchestrates executor + server with shared `data/keys` volume

**Storage Backends**:
- Main chain: LevelDB (`internal/storage/leveldb_multichain.go`)
- Upgrade state: BoltDB (`consensus/upgrade/boltdb_persistence.go`)
- Message cache: LevelDB (`internal/storage/leveldb_message_cache.go`)
- Multi-chain: supports parallel main + N candidate chains

**Web API Surface** (`internal/apis/handlers/upgrade_handler.go`):
- Upgrade: `/api/upgrade/propose`, `/api/upgrade/start`, `/api/upgrade/rollback`, `/api/upgrade/status`
- Candidates: `/api/candidate/start`, `/api/candidate/list`, `/api/candidate/:id/state`
- Metrics: `/api/metrics/current`, `/api/metrics/history`
- Health: `/api/health`
- All return JSON via Gin framework (`model.ResponseData` wrapper)

## Typical tasks (how AI should approach code changes)

1. Before changing public APIs or proto files: run `make compile_proto` and `make build-cmds` locally (or in CI) to ensure codegen and builds succeed.
2. For runtime changes touching keys/config: verify `data/keys/` exists and regenerate keys with `make genkey` if tests expect them.
3. When adding binaries, follow the `cmd/<name>/main.go` pattern and add a `build-<name>` target or rely on `build-cmds`.
4. If you change networking code in `p2p/`, validate local `go.mod` `replace` paths and run `make test`.

## Quick contract for automated edits

- Inputs: file modifications under `cmd/`, `consensus/`, `pkg/proto/`, or `p2p/`.
- Outputs: `bin/` binaries built via `make build` or `make build-cmds`; regenerated proto sources under `pkg/proto/`.
- Error modes: build failures (Go compile error), proto codegen mismatch, missing `data/keys/` at runtime.

## Notable edge-cases / gotchas

**Go Module Quirks**:
- `go.mod` specifies `go 1.22`, but Dockerfile uses `golang:1.22` (builder) - version must stay aligned
- Local `replace` directives are required; refactoring `p2p-adaptor` or `blockchain-crypto` requires updating paths
- Module path is `github.com/zzz136454872/upgradeable-consensus` - respect this in all imports

**Configuration Files**:
- Three config variants: `config.yaml` (dev), `config_docker.yaml` (container), `config_prod.yaml` (production)
- **Port conflicts**: Server uses 6060 (P2P), 7070 (RPC), 8088 (HTTP). Executor uses 9877. Ensure availability.
- Node count in config must match generated keys (`make run_genkey` generates 3 by default)

**Data Directory Structure**:
- `data/keys/` - threshold signature keys (regenerate with `make run_genkey`)
- `data/node-{id}/` - per-node blockchain data (created at runtime)
- `data/network-{id}/` - network state for multi-consensus scenarios
- **Test cleanup**: Tests use `t.TempDir()`, but manual runs may leave data in `data/`

**Build Dependencies**:
- Protobuf: requires `protoc-gen-go` and `protoc-gen-go-grpc` in `$PATH`
- Keys: many tests/runtime scenarios fail without `data/keys/*` - run `make run_genkey` first
- Web: frontend build (`make build_web`) requires Node.js 18+, separate from Go toolchain

**API/Service Lifecycle**:
- Server must start BEFORE executor (executor connects via gRPC)
- `docker-compose.yml` handles this with `depends_on: executor`, but manual starts need ordering
- Upgrade Manager expects persistence layer (`BoltDBPersistence`) - tests must call `NewUpgradeManagerWithPersistence`

**Networking Assumptions**:
- P2P layer supports two modes: libp2p (`p2p-adaptor`) or simple TCP (`p2p/server.go`)
- Config `address_ip` defaults to `127.0.0.1` - multi-machine setups need external IPs
- Topic-based pub/sub: all nodes must subscribe to same `config.Topic` (set in YAML)

**Logging & Debugging**:
- `pprof` server runs on port 6061 in `main.go` - profiling interferes with port 6060 (P2P)
- Logs use structured format `[ConsensusType]\tmessage` - grep-friendly but tab-delimited
- Test logs at `logrus.InfoLevel` by default; set `DebugLevel` for verbose output

**Docker Build Context**:
- Dockerfile copies entire `.` - large `data/` directories slow builds
- `.dockerignore` recommended for `data/`, `bin/`, `web/node_modules/`
- Tsinghua APT mirrors hardcoded - international users should sed-replace or remove

## Where to look for examples in this repo

**Build & Deployment**:
- Makefile targets: `makefile` (complete build automation reference)
- Docker multi-stage: `Dockerfile` (executor + server split builds)
- Compose orchestration: `docker-compose.yml` (service dependencies)

**Core Entry Points**:
- Consensus node: `cmd/server/server.go` (main service entry)
- Executor service: `cmd/executor/main.go` (transaction execution)
- Demo launcher: `main.go` (multi-node simulation)
- Key generation: `cmd/genkey/main.go` (threshold key setup)

**Consensus Implementations**:
- POT engine: `consensus/pot/engine.go` (VDF-based consensus)
- HotStuff variants: `consensus/hotstuff/{basic,chained,eventdriven}/`
- POW: `consensus/pow/engine.go`
- Whirly: `consensus/whirly/`

**Upgrade System**:
- Manager: `consensus/upgrade/manager.go` (orchestrates 7-phase workflow)
- Multi-chain: `consensus/upgrade/multi_chain.go` (parallel chain execution)
- Persistence: `consensus/upgrade/boltdb_persistence.go` (state recovery)
- Factory: `consensus/upgrade/consensus_factory.go` (dynamic consensus instantiation)

**Testing Patterns**:
- E2E upgrade: `tests/upgrade/consensus_switch_test.go` (complete workflow demo)
- API integration: `internal/apis/handlers/upgrade_integration_test.go`
- Unit tests: `consensus/pot/pot_test.go`, `consensus/upgrade/*_test.go`
- Test helpers: `tests/upgrade/consensus_switch_test.go` has `newMockConsensus`, `createTestBlock`

**API & Handlers**:
- Handler pattern: `internal/apis/handlers/upgrade_handler.go` (RegisterRoutes example)
- Service model: `internal/apis/model/upgrade_service.go` (interface definition)
- Router setup: `internal/apis/apis.go` (Gin initialization)

**Configuration**:
- Schema: `config/config.go` (ConsensusConfig, ReplicaInfo structs)
- YAML examples: `config/config.yaml` (dev), `config_docker.yaml` (container)
- Multi-consensus: `config/config.yaml` shows HotStuff/POW/POT/Whirly sections

**Protobuf Definitions**:
- Consensus messages: `pkg/proto/hotstuff.proto`, `pkg/proto/pot.proto`
- Execution protocol: `pkg/proto/executor.proto`
- Common types: `pkg/proto/common.proto` (Block, QuorumCert, Request)

**Web Frontend**:
- Component structure: `web/src/components/`
- API integration: `web/src/api/` (HTTP client + WebSocket)
- Mock data: `web/src/mock/` (standalone development)
- README: `web/README.md` (setup & architecture)

## After changes

- Run `make test` and `make build-cmds`. If you modified protos, run `make compile_proto` first.
- If your change touches startup/keys/config, run `make genkey` and test `make run_server` + `make run_executor` locally.

If any section is unclear or you'd like the agent to include example code transforms (refactors, proto changes, or a CI patch), tell me which area to expand.
