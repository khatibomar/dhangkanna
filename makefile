.PHONY: build-frontend
build-frontend:
	tsc --outDir ./static game.ts

.PHONY: build-backend
build-backend:
	go build -o dhangkanna

.PHONY: clean
clean:
	rm -f ./static/game.js
	rm -f dhangkanna

.PHONY: build
build: build-frontend build-backend
