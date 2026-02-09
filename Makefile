.PHONY: build run test templ css dev docker-dev docker-prod clean

build: templ css
	go build -o bin/family-hub .

run: build
	./bin/family-hub

templ:
	templ generate

css:
	npx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify

dev:
	air

docker-dev:
	docker compose up --build

docker-prod:
	docker compose -f docker-compose.prod.yml up --build

test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

clean:
	rm -rf bin/ tmp/ static/css/styles.css
