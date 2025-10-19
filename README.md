
## Project overview

This repository contains a reference implementation of an Upgradeable Consensus framework. It is intended for research and demonstration of upgradeable consensus designs, interactions between consensus components, and integration with executors. The codebase is written in Go and includes consensus cores, networking, p2p layers, protobuf definitions, and example commands.

This README follows common software engineering conventions: overview, prerequisites, quick start (build/generate keys/run/test), development instructions (proto generation, debugging), directory notes, and troubleshooting.

## Featchers
以pot共识为基础，支持共识热切换。当前已支持共识包括：
- hotstuff[basic,event-driven,chained]
- pow
- pot
- whirly
- pos[todo]

## High-level layout

- `cmd/` - entry points for different executables (server, client, genkey, etc.).
- `pb/` - protobuf definitions and generated Go files.
- `consensus/`, `model/`, `types/`, `utils/` - consensus implementations, data models and utility code.
- `p2p/`, `node/` - networking and node-level code.
- `keys/` - runtime/test key files (created by `make genkey`).
- `bin/` - default output directory for binaries produced by `make`.

(See repository root for the full structure.)

## Prerequisites

Before you start, ensure the following tools are installed and available on your PATH:

- Go 1.20+ (recommended)
- make
- protoc (Protocol Buffers compiler)
- protoc-gen-go and protoc-gen-go-grpc (protobuf/grpc code generators for Go)

To install the Go protobuf plugins:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Confirm that `go`, `protoc`, `protoc-gen-go` and `protoc-gen-go-grpc` are accessible from your PATH.

## Quick start (local example)

Run these commands from the repository root.

1) Build all binaries (output to `bin/`):

```bash
make build
```

2) Generate example keys (written into `keys/`):

```bash
make genkey
```

3) Run tests:

```bash
make test
```

4) Start the example server (open a terminal for this):

```bash
make run_server
```

5) Start the mock executor or remote executor (open another terminal):

```bash
make run_executor
```

Note: some commands log to `logs/`. You can also run binaries directly from `bin/` with arguments for more targeted testing.

## Protobuf code generation

If you modify `pb/*.proto`, regenerate the Go sources:

```bash
make compile_proto
```

The Makefile embeds plugin options into `--go-grpc_out`. If you prefer to run `protoc` manually, for example:

```bash
protoc --go_out=. --go-grpc_out=. -I pb pb/*.proto
```

Make sure `protoc-gen-go` and `protoc-gen-go-grpc` are available in your PATH.

## Development and debugging

- Use the `.vscode/launch.json` configurations (located in the repo root) for debugging in VS Code. Those configurations set `cwd` to the project root.
- Select a configuration in the Run & Debug panel (for example, "Launch server") and start debugging.
- To step through a specific package, place breakpoints in the relevant `cmd/*` `main.go` and launch the associated debug configuration.

## Useful Makefile targets

- `make` — build primary binaries into `bin/`.
- `make build-cmds` — build all commands under `cmd/`.
- `make genkey` — build and run key generator; writes keys to `keys/`.
- `make test` — run Go tests.
- `make compile_proto` — generate Go code from `pb/*.proto`.
- `make run_server` — run the server binary (demo).
- `make run_remote` — run a mock remote executor/client.
- `make clean` — remove built binaries, keys and logs (see Makefile).

## Example run sequence (recommended)

1. `make buiild` to build.
2. `make genkey` (only once or when regenerating keys).
3. In terminal A: `make run_server`.
4. In terminal B: `make run_remote` or `bin/remote`.

## Key files and important sources

- `main.go` (repo root): top-level entry or demo launcher; it may call into different commands.
- `cmd/`: individual CLI entrypoints (server, client, genkey, ...).
- `pb/`: proto definitions and generated artifacts (keep generator versions compatible with `go.mod`).
- `consensus/`: consensus implementations (hotstuff, pot, pow, etc.).

## Troubleshooting

- "make: go: No such file or directory": install Go and ensure `go` is on PATH (`go version`).
- "ERROR: Invalid or corrupt Go version": your Go installation may be corrupted. Reinstall following https://go.dev/doc/install and confirm `go version`.
- Protobuf generation fails: verify `protoc` and `protoc-gen-go`/`protoc-gen-go-grpc` are installed and their binaries are on PATH; ensure plugin versions are compatible with `go.mod`.
- Runtime cannot find keys: check `keys/` contains the expected `.key` files or run `make genkey`.

If you'd like, I can:

- run `make -n help` and print the Makefile targets (this just prints commands without executing them), so you can confirm available targets;
- or create a focused developer guide for a submodule (for example, `consensus/pot`).

## License

This project is licensed under the terms in the `LICENSE` file at the repository root.

## Acknowledgements

Thanks to contributors and the open-source projects that provided tools and reference implementations.

---

If you want me to run `make -n help` now or generate per-module documentation, tell me which target or module and I'll proceed.

