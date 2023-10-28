build: build_genkey build_client build_server
	@:

build_genkey:
	go build -o genkey ./cmd/genkey

build_client:
	go build -o client ./cmd/client

build_server:
	go build -o upgradeable-consensus ./cmd/server

win_bulid:
	go build -o genkey.exe ./cmd/genkey
	go build -o client.exe ./cmd/client
	go build -o upgradeable-consensus.exe ./cmd/server

test:
	go clean -testcache
	go test -v ./...

compile_proto:
	protoc pb/*.proto --go_out=. --go-grpc_out=.

win_compile_proto:
	protoc --go_out=plugins=grpc:. pb/*.proto

run_server:
	./upgradeable-consensus

run_client:
	./client

generate_key: 
	./genkey -p keys/ -k 3 -l 4

win_generate_key: 
	./genkey.exe -p keys/ -k 3 -l 4
