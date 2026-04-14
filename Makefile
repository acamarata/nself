BINARY := nself
MODULE := github.com/nself-org/cli
DIST_DIR := dist
VERSION := $(shell cat .github/VERSION 2>/dev/null || git describe --tags 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build clean test vet install cross dist verify-prod

verify-prod:
	@bash scripts/prod-verify/p87-verification.sh

build:
	CGO_ENABLED=0 go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/nself/

install: build
	cp $(BINARY) /usr/local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	rm -rf $(DIST_DIR)

test:
	CGO_ENABLED=0 go test -mod=vendor ./...

vet:
	CGO_ENABLED=0 go vet -mod=vendor ./...

cross: cross-linux cross-darwin

cross-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-amd64 ./cmd/nself/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-arm64 ./cmd/nself/

cross-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-amd64 ./cmd/nself/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-arm64 ./cmd/nself/

dist:
	@mkdir -p $(DIST_DIR)
	@for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		name=$(BINARY)-$(VERSION)-$$os-$$arch; \
		echo "Building $$os/$$arch..."; \
		mkdir -p $(DIST_DIR)/$$name; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -mod=vendor -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$$name/$(BINARY) ./cmd/nself/; \
		cp README.md LICENSE $(DIST_DIR)/$$name/; \
		tar -czf $(DIST_DIR)/$$name.tar.gz -C $(DIST_DIR) $$name; \
		rm -rf $(DIST_DIR)/$$name; \
	done
	@cd $(DIST_DIR) && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum *.tar.gz > checksums.txt; \
	else \
		shasum -a 256 *.tar.gz > checksums.txt; \
	fi
	@echo "dist/ ready."
