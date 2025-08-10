.PHONY: build run clean dev templ server tailwind

# Build the application
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o bin/the-ark ./cmd/main.go

# Run the application
run:
	go run ./cmd/main.go

# Clean build artifacts
clean:
	rm -rf bin/ tmp/

# Run templ generation in watch mode
templ:
	templ generate --watch --proxy="http://localhost:4000" --open-browser=false

# Run air for Go hot reload
server:
	air \
	--build.cmd "go build -o tmp/bin/the-ark ./cmd/main.go" \
	--build.bin "tmp/bin/the-ark" \
	--build.delay "100" \
	--build.exclude_dir "node_modules" \
	--build.include_ext "go" \
	--build.stop_on_error "false" \
	--misc.clean_on_exit true

# Watch Tailwind CSS changes (if we add Tailwind later)
tailwind:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch

# Start development server with all watchers
dev:
	make -j2 templ server

# Generate templ files
generate:
	go tool templ generate

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Lint code
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Install templUI components
templui-add:
	@read -p "Enter component name(s): " components; \
	templui add $$components

# Show help
help:
	@echo "Available commands:"
	@echo "  build      - Build the application"
	@echo "  run        - Run the application"
	@echo "  dev        - Start development server with hot reload"
	@echo "  templ      - Watch templ files only"
	@echo "  server     - Run Go server with Air only"
	@echo "  generate   - Generate templ files"
	@echo "  deps       - Install dependencies"
	@echo "  test       - Run tests"
	@echo "  lint       - Lint code"
	@echo "  fmt        - Format code"
	@echo "  clean      - Clean build artifacts"
	@echo "  templui-add - Add templUI components"
