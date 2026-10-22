.PHONY: build run test test-coverage dev docker-dev docker-prod clean

build:
	go build -o bin/family-hub .

run: build
	./bin/family-hub

dev:
	go run github.com/air-verse/air@latest

docker-dev:
	docker compose up --build

docker-prod:
	docker compose -f docker-compose.prod.yml up --build

test:
	@GOTESTSUM=$(shell go env GOPATH)/bin/gotestsum; \
	if [ ! -x "$$GOTESTSUM" ]; then \
		go install gotest.tools/gotestsum@latest; \
	fi; \
	"$$GOTESTSUM" -- -v ./...

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	rm -rf bin/ tmp/ coverage.out
