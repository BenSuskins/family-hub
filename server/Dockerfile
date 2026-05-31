FROM oven/bun:1-alpine AS css-builder
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY tailwind.config.js ./
COPY templates/ ./templates/
COPY static/css/input.css ./static/css/input.css
RUN bunx tailwindcss -i ./static/css/input.css -o ./static/css/styles.css --minify

FROM golang:1.25-alpine AS go-builder
WORKDIR /app
RUN go install github.com/a-h/templ/cmd/templ@v0.3.977
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -o family-hub .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /app/family-hub .
COPY --from=css-builder /app/static/css/styles.css ./static/css/styles.css
COPY static/js/ ./static/js/

EXPOSE 8080
CMD ["./family-hub"]
