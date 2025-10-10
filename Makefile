NAME=jenkins-job-cli
VERSION=$(shell cat VERSION)
BUILD=$(shell git rev-parse --short HEAD)
EXT_LD_FLAGS="-Wl,--allow-multiple-definition"
LD_FLAGS="-s -w -X main.version=$(VERSION) -X main.build=$(BUILD) -extldflags=$(EXT_LD_FLAGS)"

# Default target
all: build

# Clean build artifacts
clean:
	rm -rf _build/ release/ dist/

# Build for current platform
build:
	go mod tidy
	CGO_ENABLED=0 go build -tags release -ldflags $(LD_FLAGS) -o $(NAME)
	cp $(NAME) jj  # 创建符号链接

# Development build
build-dev:
	go build -ldflags "-w -X main.version=$(VERSION)-dev -X main.build=$(BUILD) -extldflags=$(EXT_LD_FLAGS)"
	cp $(NAME) jj  # 创建符号链接

# Build for all platforms (legacy, kept for compatibility)
build-all: clean
	@echo "Building for all platforms..."
	mkdir -p _build
	GOOS=darwin  GOARCH=arm64 go build -tags release -ldflags $(LD_FLAGS) -o _build/$(NAME)-$(VERSION)-darwin-arm64
	GOOS=darwin  GOARCH=amd64 go build -tags release -ldflags $(LD_FLAGS) -o _build/$(NAME)-$(VERSION)-darwin-amd64
	GOOS=linux   GOARCH=amd64 go build -tags release -ldflags $(LD_FLAGS) -o _build/$(NAME)-$(VERSION)-linux-amd64
	GOOS=linux   GOARCH=arm   go build -tags release -ldflags $(LD_FLAGS) -o _build/$(NAME)-$(VERSION)-linux-arm
	GOOS=linux   GOARCH=arm64 go build -tags release -ldflags $(LD_FLAGS) -o _build/$(NAME)-$(VERSION)-linux-arm64
	GOOS=windows GOARCH=amd64 go build -tags release -ldflags $(LD_FLAGS) -o _build/$(NAME)-$(VERSION)-windows-amd64
	cd _build; shasum -a 256 * > sha256sums.txt
	@echo "Build completed."

# Docker image
image:
	docker build -t $(NAME) -f Dockerfile .

# Install to local system
install: build
	@echo "Installing $(NAME) to /usr/local/bin/..."
	sudo cp $(NAME) /usr/local/bin/
	sudo cp jj /usr/local/bin/  # 同时安装 jj 命令
	@echo "Installation completed. You can use 'jenkins-job-cli' or 'jj'"

# Test GoReleaser configuration
goreleaser-check:
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "Error: goreleaser is required but not installed."; \
		echo "Install it from: https://goreleaser.com/install/"; \
		echo "Or run: brew install goreleaser"; \
		exit 1; \
	}
	goreleaser check

# Test build with GoReleaser (no release)
goreleaser-snapshot: clean goreleaser-check
	goreleaser release --snapshot --clean

# Full release with GoReleaser (includes Homebrew tap)
release: clean goreleaser-check
	@echo "Releasing version $(VERSION) with GoReleaser..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		if command -v gh >/dev/null 2>&1; then \
			export GITHUB_TOKEN=$$(gh auth token); \
			echo "Using GitHub token from gh CLI"; \
		else \
			echo "Error: GITHUB_TOKEN is not set and gh CLI is not available."; \
			echo "Please set it: export GITHUB_TOKEN=ghp_xxx"; \
			echo "Or install gh: brew install gh && gh auth login"; \
			exit 1; \
		fi \
	fi
	goreleaser release --clean

# Initialize GoReleaser configuration
goreleaser-init:
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "Error: goreleaser is required but not installed."; \
		echo "Install it from: https://goreleaser.com/install/"; \
		exit 1; \
	}
	goreleaser init

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build for current platform (creates jj alias)"
	@echo "  build-dev          - Development build"
	@echo "  build-all          - Build for all platforms (legacy)"
	@echo "  release            - Release with GoReleaser + Homebrew tap (main target)"
	@echo "  goreleaser-snapshot- Test build with GoReleaser"
	@echo "  goreleaser-check   - Check GoReleaser configuration"
	@echo "  install            - Install to /usr/local/bin/ (both jenkins-job-cli and jj)"
	@echo "  clean              - Clean build artifacts"

.PHONY: all clean build build-dev build-all image install goreleaser-check goreleaser-snapshot release goreleaser-init help