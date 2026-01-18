VERSION := 1.0.0
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms"
	@echo "  install     - Install to /usr/local/bin"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  run         - Run with test config"

.PHONY: build
build:
	@echo "Building cloudm-cli..."
	go build -ldflags "$(LDFLAGS)" -o bin/cloudm-cli main.go
	@echo "Done! Binary: bin/cloudm-cli"

.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/cloudm-cli-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/cloudm-cli-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/cloudm-cli-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/cloudm-cli-windows-amd64.exe main.go
	@echo "Done! Binaries in bin/"

.PHONY: install
install: build
	@echo "Installing to /usr/local/bin/cloudm-cli..."
	sudo cp bin/cloudm-cli /usr/local/bin/cloudm-cli
	@echo "Done!"

.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf migrations/
	rm -f *.dump *.log *.txt
	@echo "Done!"

.PHONY: run
run: build
	./bin/cloudm-cli --help

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

.PHONY: release
release:
	@read -p "Enter version (e.g., v1.0.0): " version; \
	git tag -a $$version -m "Release $$version"; \
	git push origin $$version; \
	echo "Released $$version - GitHub Actions will build and publish"

.PHONY: release-local
release-local:
	@echo "Building release binaries locally..."
	@./.github/workflows/scripts/build-release.sh