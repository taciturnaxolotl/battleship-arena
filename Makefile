.PHONY: build run clean test docker-build docker-run help

# Build the battleship arena server
build:
	@echo "Building battleship-arena..."
	@go build -o battleship-arena

# Run the server
run: build
	@echo "Starting battleship-arena..."
	@./battleship-arena

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f battleship-arena
	@rm -rf submissions/ .ssh/ *.db

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Generate SSH host key
gen-key:
	@echo "Generating SSH host key..."
	@mkdir -p .ssh
	@ssh-keygen -t ed25519 -f .ssh/battleship_arena -N ""

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Update dependencies
deps:
	@echo "Updating dependencies..."
	@go mod tidy
	@go mod download

# Build for production (optimized)
build-prod:
	@echo "Building for production..."
	@CGO_ENABLED=1 go build -ldflags="-s -w" -o battleship-arena

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the server"
	@echo "  run         - Build and run the server"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  gen-key     - Generate SSH host key"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  deps        - Update dependencies"
	@echo "  build-prod  - Build optimized production binary"
	@echo "  help        - Show this help"
