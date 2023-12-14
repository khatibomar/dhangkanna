ifeq ($(OS),Windows_NT)
	EXECUTABLE_BACKEND := dhangkanna_back.exe
	EXECUTABLE_FRONTEND := dhangkanna_front.exe
else
	EXECUTABLE_BACKEND := dhangkanna_back
	EXECUTABLE_FRONTEND := dhangkanna_front
endif

.PHONY: build-frontend
build-frontend:
	tsc --outDir ./cmd/frontend/static ./cmd/frontend/game.ts
	go build -o $(EXECUTABLE_FRONTEND) ./cmd/frontend

.PHONY: build-backend
build-backend:
	go build -o $(EXECUTABLE_BACKEND) ./cmd/api

.PHONY: clean
clean:
	$(if $(filter Windows%,$(OS)),del /Q .\cmd\frontend\static\game.js,rm -f ./cmd/frontend/static/game.js)
	$(if $(filter Windows%,$(OS)),del /Q $(EXECUTABLE_BACKEND),rm -f $(EXECUTABLE_BACKEND))
	$(if $(filter Windows%,$(OS)),del /Q $(EXECUTABLE_FRONTEND),rm -f $(EXECUTABLE_FRONTEND))
	$(if $(filter Windows%,$(OS)),del /Q /F /S /A .\cmd\api\v1\game.pb.go,rm -f ./cmd/api/v1/game.pb.go)
	$(if $(filter Windows%,$(OS)),del /Q /F /S /A .\cmd\api\v1\game_grpc.pb.go,rm -f ./cmd/api/v1/game_grpc.pb.go)

.PHONY: build
build: proto build-frontend build-backend

PHONY: proto
proto:
		protoc cmd/api/v1/*.proto \
				--go_out=. \
				--go-grpc_out=. \
				--go_opt=paths=source_relative \
				--go-grpc_opt=paths=source_relative \
				--proto_path=.

## nodes
.PHONY: node1
node1:
	./$(EXECUTABLE_BACKEND) -bootstrap -data-dir="/tmp/dhangkanna/node1" -node-name="node1"

.PHONY: node2
node2:
	./$(EXECUTABLE_BACKEND) -data-dir="/tmp/dhangkanna/node2" -node-name="node2" -bind-addr="127.0.0.1:7001" -rpc-port=7002 -start-join-addrs="127.0.0.1:4001"

.PHONY: node3
node3:
	./$(EXECUTABLE_BACKEND) -data-dir="/tmp/dhangkanna/node3" -node-name="node3" -bind-addr="127.0.0.1:8001" -rpc-port=8002 -start-join-addrs="127.0.0.1:4001"
