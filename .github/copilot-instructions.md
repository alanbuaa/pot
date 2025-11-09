## Purpose

This file gives concise, repo-specific guidance for AI coding agents (Copilot/automation) so they can be productive quickly. It focuses on the architecture, build/test/run workflows, project-specific conventions, integration points and concrete examples from this codebase.

## Big-picture architecture

- Multi-binary Go repository. Primary entry points live under `cmd/` (e.g. `cmd/server`, `cmd/client`, `cmd/genkey`, `cmd/pot_test`). Building produces binaries into `bin/`.
- Consensus implementations live in `consensus/` (hotstuff, pot, pow, whirly, ...). Higher-level orchestration and models are in `model/` and `types/`.
- Networking & P2P: `p2p/` and `p2p/p2p-adaptor/` contain the networking adapters. `go.mod` uses `replace` directives to map local packages (see `replace p2padaptor => ./p2p/p2p-adaptor`).
- Protobufs: `.proto` files are under `pkg/proto/` and generated Go code lives beside them. Use the Makefile target to regenerate.
- Config: YAML config files live in `config/` and are parsed by `config.go`.

Key file examples: `main.go` (repo root) spawns demo nodes; `cmd/server/server.go` implements the server binary; `consensus/` contains algorithm implementations.

## Build / test / run (concrete commands)

- Build all primary binaries: `make build` (uses `go build` and writes to `bin/`).
- Build all commands under `cmd/`: `make build-cmds`.
- Generate keys (used by runtime/tests): `make genkey` or `make run_genkey` (writes keys to `data/keys/` by default).
- Run the demo server: `make run_server` (builds then executes `bin/server`).
- Run the remote/mock executor/client: `make run_remote` / `make run_executor`.
- Regenerate protobufs: `make compile_proto` (runs `protoc -I pkg/proto pkg/proto/*.proto` with Makefile flags).
- Run tests: `make test` (expands to `go test -v ./...`).

Notes: prefer the Makefile targets (they wrap build flags, output locations and codegen flags used across the project).

## Project-specific conventions & patterns

- Binaries go to `bin/`. Use `make` targets rather than ad-hoc `go build` to ensure consistent output paths and flags.
- Keys and test data: `data/keys/` (and previously `keys/` in docs). `make genkey` is the canonical way to produce required key files.
- Protobuf plugin flags are controlled by the Makefile (`PROTO_FLAGS`). Avoid manual `protoc` calls unless matching those flags.
- The repository module path in `go.mod` is `github.com/zzz136454872/upgradeable-consensus`; local replacements exist. When editing imports or module paths, update `go.mod` `replace` directives if needed.
- Logging/pipeline: the repo uses a central logger (`pkg/logging`) and imports a `pprof` hook in `main.go` for profiling.

## Integration points & external dependencies

- `pkg/proto/*` — change proto files only when you will also run `make compile_proto` and update generated files.
- `p2p/p2p-adaptor` and local `blockchain-crypto` are wired via `replace` in `go.mod` for local development. Do not break those relative paths.
- Dockerfile: uses `golang:1.20` and sets `GOPROXY` to `https://goproxy.cn,direct`; it also switches APT mirrors to Tsinghua. Keep this in mind when changing container build base images.

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

- `go.mod` uses `replace` for local modules; automated refactors that move files across these packages require updating `replace` or paths.
- README mentions `.vscode/launch.json` for debugging, but that file is not present in the repo—do not assume VS Code launch configs exist.
- Dockerfile mutates Debian APT mirrors (to Tsinghua). CI or container builds in other environments may need adjustment.
- Go version: `go.mod` indicates `go 1.22`, but Dockerfile uses `golang:1.20`. Be cautious when changing language features or CI that assumes a particular Go version.

## Where to look for examples in this repo

- Building/running patterns: `makefile` (root)
- Entry points: `cmd/server/`, `cmd/client/`, `cmd/genkey/` and `main.go` (root)
- Consensus implementations: `consensus/` and `model/pot/`
- Protobufs: `pkg/proto/`
- P2P adapters and local module replacements: `p2p/p2p-adaptor/` and `go.mod`

## After changes

- Run `make test` and `make build-cmds`. If you modified protos, run `make compile_proto` first.
- If your change touches startup/keys/config, run `make genkey` and test `make run_server` + `make run_executor` locally.

If any section is unclear or you'd like the agent to include example code transforms (refactors, proto changes, or a CI patch), tell me which area to expand.
