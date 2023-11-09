ifeq ($(OS),Windows_NT)
	EXECUTABLE_BACKEND := dhangkanna_back.exe
	EXECUTABLE_FRONTEND := dhangkanna_front.exe
else
	EXECUTABLE_BACKEND := dhangkanna_back.exe
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