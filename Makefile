BINARY := nself
MODULE := github.com/nself-org/cli
DIST_DIR := dist
VERSION := $(shell cat .github/VERSION 2>/dev/null || git describe --tags 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
NSELF_LICENSE_PUBKEY_HEX ?=
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE) \
	-X $(MODULE)/internal/license.licensePubKeyHex=$(NSELF_LICENSE_PUBKEY_HEX)
BUILDFLAGS := -trimpath

.PHONY: build clean test vet install cross dist verify-prod sport-f21 sbom man

verify-prod:
	@bash scripts/prod-verify/p87-verification.sh

sport-f21:
	@bash scripts/sport/generate-f21.sh

## Q04 — SBOM generation (local dev target)
## Requires: syft (https://github.com/anchore/syft)
## Generates sbom.spdx.json (SPDX) and sbom.cdx.json (CycloneDX 1.5) from the source tree.
## Run `make sbom` before a release to verify SBOM generation works locally.
sbom:
	@echo "Checking for syft..."
	@if ! command -v syft >/dev/null 2>&1; then \
		echo "syft not found. Install via:"; \
		echo "  curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"; \
		exit 1; \
	fi
	@echo "Generating SPDX SBOM..."
	@syft packages . --output spdx-json=sbom.spdx.json
	@echo "Generating CycloneDX 1.5 SBOM (Q04)..."
	@syft packages . --output cyclonedx-json=sbom.cdx.json
	@echo ""
	@echo "SBOMs written:"
	@echo "  sbom.spdx.json  (SPDX 2.3)        $$(wc -c < sbom.spdx.json) bytes"
	@echo "  sbom.cdx.json   (CycloneDX 1.5)   $$(wc -c < sbom.cdx.json) bytes"
	@echo ""
	@echo "Verify a release SBOM signature:"
	@echo "  bash tools/sbom/verify.sh v$(VERSION)"
	@echo ""
	@echo "Query SBOM for a package:"
	@echo "  bash tools/sbom/query.sh --local sbom.cdx.json --pkg cobra"

## man — generate man pages for all nself commands into ./man/
man: build
	@mkdir -p man
	@./$(BINARY) man --output man
	@echo "Man pages written to man/"

build:
	CGO_ENABLED=0 go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY) ./cmd/nself/

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
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-amd64 ./cmd/nself/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-linux-arm64 ./cmd/nself/

cross-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-amd64 ./cmd/nself/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(BINARY)-darwin-arm64 ./cmd/nself/

dist:
	@mkdir -p $(DIST_DIR)
	@for platform in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		name=$(BINARY)-$(VERSION)-$$os-$$arch; \
		echo "Building $$os/$$arch..."; \
		mkdir -p $(DIST_DIR)/$$name; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$$name/$(BINARY) ./cmd/nself/; \
		cp README.md LICENSE $(DIST_DIR)/$$name/; \
		tar -czf $(DIST_DIR)/$$name.tar.gz -C $(DIST_DIR) $$name; \
		rm -rf $(DIST_DIR)/$$name; \
	done
	@for arch in amd64 arm64; do \
		name=$(BINARY)-$(VERSION)-windows-$$arch; \
		echo "Building windows/$$arch..."; \
		mkdir -p $(DIST_DIR)/$$name; \
		CGO_ENABLED=0 GOOS=windows GOARCH=$$arch go build $(BUILDFLAGS) -mod=vendor -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$$name/$(BINARY).exe ./cmd/nself/; \
		cp README.md LICENSE $(DIST_DIR)/$$name/; \
		cd $(DIST_DIR) && zip -qr $$name.zip $$name && cd ..; \
		rm -rf $(DIST_DIR)/$$name; \
	done
	@cd $(DIST_DIR) && if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum *.tar.gz *.zip > checksums.txt; \
	else \
		shasum -a 256 *.tar.gz *.zip > checksums.txt; \
	fi
	@echo "dist/ ready."
