.PHONY: build test test-integration clean install lint fmt vet

# Variables
BINARY_NAME := ccpersona
MAIN_PATH := ./cmd
BUILD_DIR := ./dist
INSTALL_PATH := $(GOPATH)/bin

# Build flags
LDFLAGS := -ldflags "-X main.version=$$(git describe --tags --always) -X main.revision=$$(git rev-parse --short HEAD)"

# Default target
all: clean build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Build for all platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

# Run tests (skip integration tests that require external services)
test:
	@echo "Running tests..."
	@go test -v -short -race -coverprofile=coverage.out ./...

# Run all tests including integration tests
test-integration:
	@echo "Running all tests including integration..."
	@go test -v -race -coverprofile=coverage.out ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@cp $(BINARY_NAME) $(INSTALL_PATH)/

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run

# Run all checks
check: fmt vet test

# Development mode with hot reload (requires air)
dev:
	@air

# Generate mocks (if needed)
generate:
	@echo "Generating code..."
	@go generate ./...

# Build a snapshot with goreleaser (requires goreleaser)
snapshot:
	@echo "Building snapshot with goreleaser..."
	@goreleaser release --snapshot --clean

# Test release build locally (requires goreleaser)
release-test:
	@echo "Testing release build..."
	@goreleaser release --skip=publish --clean

# Create a new release tag
tag:
	@echo "Current tags:"
	@git tag -l
	@echo ""
	@read -p "Enter new version (e.g., v1.0.0): " version; \
	git tag -a $$version -m "Release $$version"
	@echo "Tag created. Run 'git push origin --tags' to trigger the release."