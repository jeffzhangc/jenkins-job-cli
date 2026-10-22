NAME=jenkins-job-cli
VERSION=$(shell cat VERSION)
BUILD=$(shell git rev-parse --short HEAD)
EXT_LD_FLAGS="-Wl,--allow-multiple-definition"
LD_FLAGS="-s -w -X main.version=$(VERSION) -X main.build=$(BUILD) -extldflags=$(EXT_LD_FLAGS)"

# 交叉打包平台列表: <GOOS>/<GOARCH>
PLATFORMS := darwin/arm64 darwin/amd64 linux/arm64 linux/amd64 windows/arm64 windows/amd64

# sha256 工具: macOS 用 shasum, Linux 用 sha256sum
SHASUM := $(shell command -v shasum >/dev/null 2>&1 && echo "shasum -a 256" || echo "sha256sum")

# Default target
all: build

# Clean build artifacts
clean:
	rm -rf _build/ release/ dist/

# Build for current platform
build:
	@mkdir -p dist
	go mod tidy
	CGO_ENABLED=0 go build -tags release -ldflags $(LD_FLAGS) -o dist/$(NAME)
	ln -sf $(NAME) dist/jj  # 创建符号链接
	@bash ./scripts/completions.sh

build-dev:
	@mkdir -p dist
	go build -ldflags "-w -X main.version=$(VERSION)-dev -X main.build=$(BUILD) -extldflags=$(EXT_LD_FLAGS)" -o dist/$(NAME)
	ln -sf $(NAME) dist/jj  # 创建符号链接

# Build for all platforms and package distributable archives:
#   Linux / macOS -> .tar.gz    Windows -> .zip
# 产物输出到 dist/, 每个包内含二进制、README、LICENSE 和 shell 补全脚本
package:
	@command -v zip >/dev/null 2>&1 || { \
		echo "Error: 'zip' is required to package Windows builds."; \
		echo "Install it (e.g. brew install zip / apt install zip) and retry."; \
		exit 1; \
	}
	@bash ./scripts/completions.sh
	@mkdir -p dist
	@rm -f dist/$(NAME)-$(VERSION)-*.tar.gz dist/$(NAME)-$(VERSION)-*.zip
	@set -e; for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		pkg=$(NAME)-$(VERSION)-$$os-$$arch; \
		staging=dist/staging/$$pkg; \
		echo "==> Building $$pkg ..."; \
		rm -rf $$staging; mkdir -p $$staging/completions; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; else ln -sf $(NAME) $$staging/jj; fi; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -tags release -ldflags $(LD_FLAGS) -o $$staging/$(NAME)$$ext; \
		cp README.md README_CN.md LICENSE $$staging/; \
		cp completions/jj.bash completions/jj.zsh completions/jj.fish $$staging/completions/; \
		if [ "$$os" = "windows" ]; then \
			(cd dist/staging && zip -qr ../$$pkg.zip $$pkg); \
			echo "    -> dist/$$pkg.zip"; \
		else \
			tar -czf dist/$$pkg.tar.gz -C dist/staging $$pkg; \
			echo "    -> dist/$$pkg.tar.gz"; \
		fi; \
	done
	@rm -rf dist/staging
	@cd dist && $(SHASUM) $(NAME)-$(VERSION)-*.tar.gz $(NAME)-$(VERSION)-*.zip > sha256sums.txt
	@echo "Packages completed. See dist/ and dist/sha256sums.txt"

# Legacy alias, kept for compatibility
build-all: package

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
	sudo cp dist/$(NAME) $(HOMEBREW_PREFIX)/bin/
	sudo ln -sf $(HOMEBREW_PREFIX)/bin/$(NAME) $(HOMEBREW_PREFIX)/bin/jj
	@echo "Installation completed. You can use 'jenkins-job-cli' or 'jj'"
	@echo "Installation completed."

uninstall:
	$(eval HOMEBREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo "/usr/local"))
	@echo "Uninstalling $(NAME) and completions from $(HOMEBREW_PREFIX)"
	sudo rm -f $(HOMEBREW_PREFIX)/bin/$(NAME) $(HOMEBREW_PREFIX)/bin/jj
	sudo rm -f $(HOMEBREW_PREFIX)/etc/bash_completion.d/jj
	sudo rm -f $(HOMEBREW_PREFIX)/share/zsh/site-functions/_jj
	sudo rm -f $(HOMEBREW_PREFIX)/share/fish/vendor_completions.d/jj.fish
	@echo "Uninstallation completed."

# Check GoReleaser configuration
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
	@echo "  build              - Build for current platform (creates jj symlink)"
	@echo "  build-dev          - Development build"
	@echo "  package            - Cross-compile all platforms into dist/ (linux/darwin: .tar.gz, windows: .zip)"
	@echo "  build-all          - Alias of package (legacy)"
	@echo "  release            - Release with GoReleaser + Homebrew tap (main target)"
	@echo "  goreleaser-snapshot- Test build with GoReleaser"
	@echo "  goreleaser-check   - Check GoReleaser configuration"
	@echo "  install            - Install to Homebrew bin/ (both jenkins-job-cli and jj)"
	@echo "  uninstall          - Remove installed binary and completions"
	@echo "  clean              - Clean build artifacts"

.PHONY: all clean build build-dev package build-all image install install-completions uninstall goreleaser-check goreleaser-snapshot release goreleaser-init help
