BIN_DIR := bin
GO ?= go
PROTOC ?= protoc
PROTO_FLAGS := --go_out=. --go-grpc_out=require_unimplemented_servers=false:.

CMDS := bci client genkey governance http pot_test executor server txtest upgrade-cli

.PHONY: help build build-cmds clean test compile_proto genkey run_server run_client run_executor run_governance run_bci run_http run_txtest run_genkey run_upgrade-cli docker_build docker_run

# --------------------------------------------------
# Build targets
# --------------------------------------------------
help:
	@echo "Usage: make [target]"
	@echo "Common targets: build, build-cmds, test, compile_proto, genkey, clean, docker_build"

build: build_genkey build_client build_server btest 
	@:

# build all cmd/* programs into $(BIN_DIR)
build-cmds: $(BIN_DIR) $(addprefix build-, $(CMDS))
	@:

# generic per-cmd build target (build-<name>)
# usage: make build-foo -> builds ./cmd/foo into $(BIN_DIR)/foo
build-%: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$* ./cmd/$*

build_genkey: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/genkey ./cmd/genkey

build_client: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/client ./cmd/client

build_server: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/server ./cmd/server

btest: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/test ./cmd/pot_test

# --------------------------------------------------
# Run targets
# --------------------------------------------------
run-%: build-%
	$(BIN_DIR)/$*

run_server: build_server
	$(BIN_DIR)/server

run_server_new: btest
	$(BIN_DIR)/test 2>&1 | tee log.txt

run_client: build_client
	$(BIN_DIR)/client

run_txtest: build-txtest
	$(BIN_DIR)/txtest

run_governance: build-governance
	$(BIN_DIR)/governance

run_executor: build-executor
	$(BIN_DIR)/executor

# --------------------------------------------------
# Proto / codegen targets
# --------------------------------------------------
compile_proto:
	$(PROTOC) -I pkg/proto pkg/proto/*.proto $(PROTO_FLAGS)

# --------------------------------------------------
# Test targets
# --------------------------------------------------
test:
	$(GO) clean -testcache
	$(GO) test -v ./...

pot_test: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/pot_test.exe ./cmd/pot_test

# --------------------------------------------------
# Utilities
# --------------------------------------------------

run_genkey: build_genkey
	$(BIN_DIR)/genkey -p data/keys/ -k 3 -l 4

build_web:
	@echo "Building web frontend..."
	cd web && npm run build

docker_build:
	docker build -t pot:v1.0 .

docker_build_all: build build_web docker_build
	@echo "Built binaries, web frontend, and Docker image"

docker_build_executor:
	@echo "Building executor Docker image..."
	docker build -f Dockerfile.executor -t pot-executor:latest .

docker_build_server:
	@echo "Building server Docker image..."
	docker build -f Dockerfile.server -t pot-server:latest .

docker_build_images: docker_build_executor docker_build_server
	@echo "Built executor and server Docker images"

docker_compose_up: 
	@echo "Starting services with docker-compose..."
	docker-compose up -d

docker_compose_down:
	@echo "Stopping services with docker-compose..."
	docker-compose down

docker_compose_build:
	@echo "Building and starting services with docker-compose..."
	docker-compose up --build -d

docker_compose_logs:
	docker-compose logs -f

docker_run:
	docker run -it pot-server:latest

# --------------------------------------------------
# Windows helpers
# --------------------------------------------------
win_build: $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/genkey.exe ./cmd/genkey
	$(GO) build -o $(BIN_DIR)/client.exe ./cmd/client
	$(GO) build -o $(BIN_DIR)/server.exe ./cmd/server

win_genkey: 
	$(BIN_DIR)/genkey.exe -p data/keys/ -k 3 -l 4

win_compile_proto:
	$(PROTOC) --go_out=plugins=grpc:. pkg/proto/*.proto

# --------------------------------------------------
# Misc
# --------------------------------------------------
$(BIN_DIR):
	@if [ ! -d "$(BIN_DIR)" ]; then \
		echo "Creating $(BIN_DIR) directory..."; \
		mkdir -p $(BIN_DIR); \
	fi

clean: 
	rm -rf $(BIN_DIR)/* data/*
	@echo "Cleaned up binaries and data directories."