build: build_genkey build_client build_server build_test
	@:

build_genkey:
	go build -o genkey ./cmd/genkey

build_client:
	go build -o client ./cmd/client

build_server:
	go build -o upgradeable-consensus ./cmd/server

build_test:
	go build -o test ./cmd/pot_test

win_build:
	go build -o genkey.exe ./cmd/genkey
	go build -o client.exe ./cmd/client
	go build -o upgradeable-consensus.exe ./cmd/server

test:
	go clean -testcache
	go test -v ./...

compile_proto:
	protoc pb/*.proto --go_out=. --go-grpc_out=. --go-grpc_opt=require_unimplemented_servers=false

compile_proto_new:
	protoc pb/*.proto --go_out=. --go-grpc_out=require_unimplemented_servers=false:.

win_compile_proto:
	protoc --go_out=plugins=grpc:. pb/*.proto

run_server:
	./upgradeable-consensus

run_server_new:
	./test 2>&1 | tee log.txt

run_client:
	./client

generate_key: 
	./genkey -p keys/ -k 3 -l 4

win_generate_key: 
	./genkey.exe -p keys/ -k 3 -l 4

pot_test:
	go build -o pot_test.exe ./cmd/pot_test

docker_build:
	docker build -t pot:v1.0 .

docker_run:
	docker run -it pot:v1.0