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

.PHONY: build
build: build-frontend build-backend