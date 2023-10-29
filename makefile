ifeq ($(OS),Windows_NT)
	EXECUTABLE := dhangkanna.exe
else
	EXECUTABLE := dhangkanna
endif

.PHONY: build-frontend
build-frontend:
	tsc --outDir ./static game.ts

.PHONY: build-backend
build-backend:
	go build -o $(EXECUTABLE)

.PHONY: clean
clean:
	$(if $(filter Windows%,$(OS)),del /Q .\static\game.js,rm -f ./static/game.js)
	$(if $(filter Windows%,$(OS)),del /Q $(EXECUTABLE),rm -f $(EXECUTABLE))
	$(if $(filter Windows%,$(OS)),del /Q /F /S /A .\api\v1\state.pb.go,rm -f ./api/v1/state.pb.go)
	$(if $(filter Windows%,$(OS)),del /Q /F /S /A .\api\v1\state_grpc.pb.go,rm -f ./api/v1/state_grpc.pb.go)

.PHONY: build
build: proto build-frontend build-backend

PHONY: proto
proto:
		protoc api/v1/*.proto \
				--go_out=. \
				--go-grpc_out=. \
				--go_opt=paths=source_relative \
				--go-grpc_opt=paths=source_relative \
				--proto_path=.