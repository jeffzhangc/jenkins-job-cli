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
	CGO_ENABLED=0 go build -tags release -ldflags $(LD_FLAGS) -o jenkins-job-cli
	cp $(NAME) jj  # 创建符号链接
	@bash ./scripts/completions.sh

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

# 安装补全文件到 Homebrew 目录
install-completions: build
	@echo "Installing shell completions..."
	$(eval HOMEBREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo "/usr/local"))
	@echo "Using prefix: $(HOMEBREW_PREFIX)"
	
	install -d $(HOMEBREW_PREFIX)/etc/bash_completion.d/
	install -m 644 completions/jj.bash $(HOMEBREW_PREFIX)/etc/bash_completion.d/jj
	
	install -d $(HOMEBREW_PREFIX)/share/zsh/site-functions/
	install -m 644 completions/jj.zsh $(HOMEBREW_PREFIX)/share/zsh/site-functions/_jj
	
	install -d $(HOMEBREW_PREFIX)/share/fish/vendor_completions.d/
	install -m 644 completions/jj.fish $(HOMEBREW_PREFIX)/share/fish/vendor_completions.d/jj.fish
	
	@echo "Completions installed successfully!"

# Install to local system
install: build install-completions
	@echo "Installing $(NAME) to $(HOMEBREW_PREFIX)/bin/..."
	sudo cp $(NAME) $(HOMEBREW_PREFIX)/bin/
	sudo ln -sf $(HOMEBREW_PREFIX)/bin/$(NAME) $(HOMEBREW_PREFIX)/bin/jj
	@echo "Installation completed. You can use 'jenkins-job-cli' or 'jj'"
	@echo "Installation completed."



# 使用 GoReleaser 发布
goreleaser-release: clean
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "Error: goreleaser is required but not installed."; \
		echo "Install it from: https://goreleaser.com/install/"; \
		echo "Or run: brew install goreleaser"; \
		exit 1; \
	}
	goreleaser check

# Test build with GoReleaser (no release)
goreleaser-snapshot: clean goreleaser-check
	goreleaser release --snapshot

# Full release with GoReleaser (includes Homebrew tap)
release: clean goreleaser-check
	@echo "Releasing version $(VERSION) with GoReleaser..."
	@export GITHUB_TOKEN=$$(gh auth token) && goreleaser release

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
	@echo "  install            - Install to $(HOMEBREW_PREFIX)/bin/ (both jenkins-job-cli and jj)"
	@echo "  clean              - Clean build artifacts"

.PHONY: all clean build build-dev build-all image install goreleaser-check goreleaser-snapshot release goreleaser-init help