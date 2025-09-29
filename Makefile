NAME=jenkins-job-cli
VERSION=$(shell cat VERSION)
BUILD=$(shell git rev-parse --short HEAD)
EXT_LD_FLAGS="-Wl,--allow-multiple-definition"
LD_FLAGS="-s -w -X main.version=$(VERSION) -X main.build=$(BUILD) -extldflags=$(EXT_LD_FLAGS)"


clean:
	rm -rf _build/ release/

build:
	go mod tidy
	CGO_ENABLED=0 go build -tags release -ldflags $(LD_FLAGS) -o jenkins-job-cli

build-dev:
	go build -ldflags "-w -X main.version=$(VERSION)-dev -X main.build=$(BUILD) -extldflags=$(EXT_LD_FLAGS)"

build-all: clean
	@echo "Building for all platforms..."
	mkdir -p _build
	GOOS=darwin  GOARCH=arm64 go build -tags release -ldflags $(LD_FLAGS) -o _build/jenkins-job-cli-$(VERSION)-darwin-arm64
	GOOS=darwin  GOARCH=amd64 go build -tags release -ldflags $(LD_FLAGS) -o _build/jenkins-job-cli-$(VERSION)-darwin-amd64
	GOOS=linux   GOARCH=amd64 go build -tags release -ldflags $(LD_FLAGS) -o _build/jenkins-job-cli-$(VERSION)-linux-amd64
	GOOS=linux   GOARCH=arm   go build -tags release -ldflags $(LD_FLAGS) -o _build/jenkins-job-cli-$(VERSION)-linux-arm
	GOOS=linux   GOARCH=arm64 go build -tags release -ldflags $(LD_FLAGS) -o _build/jenkins-job-cli-$(VERSION)-linux-arm64
	GOOS=windows GOARCH=amd64 go build -tags release -ldflags $(LD_FLAGS) -o _build/jenkins-job-cli-$(VERSION)-windows-amd64
	cd _build; sha256sum * > sha256sums.txt
	@echo "Build completed."

image:
	docker build -t jenkins-job-cli -f Dockerfile .

release: build-all
	@command -v gh >/dev/null 2>&1 || { \
		echo "Error: GitHub CLI (gh) is required but not installed."; \
		echo "Install it from: https://cli.github.com/"; \
		echo "Or run: brew install gh"; \
		exit 1; \
	}
	@echo "Releasing version $(VERSION)..."
	mkdir release
	# go get github.com/progrium/gh-release/...
	cp _build/* release
	cd release; sha256sum --quiet --check sha256sums.txt
	#go run github.com/progrium/gh-release@latest create jeffzhangc/$(NAME) $(VERSION) \
	#	$(shell git rev-parse --abbrev-ref HEAD) $(VERSION)
	gh release create v${VERSION} release/* --title "v${VERSION}" --notes "Release v${VERSION}" 
	@echo "Release v$(VERSION) created."


.PHONY: build