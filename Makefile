GO ?= go
PKG := github.com/censys/cencli
BUILD_DIR ?= bin
BINARY ?= censys
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w -X $(PKG)/internal/version.Version=$(VERSION) -X $(PKG)/internal/version.Commit=$(COMMIT) -X $(PKG)/internal/version.Date=$(DATE)

# Pinned tool versions (override via env)
SQLC_VERSION ?= v1.30.0
GOIMPORTS_VERSION ?= v0.25.0
MOCKGEN_VERSION ?= v0.6.0
GOLANGCI_LINT_VERSION ?= v1.64.8
STATICCHECK_VERSION ?= 2025.1.1

# Packages to include in coverage (exclude generated mocks)
PKGS := $(shell $(GO) list ./... | grep -vE 'gen/*')

build: clean $(BINARY)

all: fmt lint build test

# One-stop local verification similar to CI (no e2e)
verify: fmt vet
	staticcheck ./...
	$(MAKE) lint
	$(MAKE) build
	$(MAKE) test-race
	$(MAKE) cover-check

tools:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION)
	go install go.uber.org/mock/mockgen@$(MOCKGEN_VERSION)
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

sqlc:
	cd internal/store/db && sqlc generate

mocks:
	@echo "Generating mocks using go:generate directives..."
	go generate ./internal/...


$(BINARY):
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY) cmd/cencli/main.go

clean:
	rm -f $(BUILD_DIR)/$(BINARY)

fmt:
	$(GO) fmt ./...
	goimports -w .

vet:
	$(GO) vet ./...

lint:
	golangci-lint run --timeout=5m

test: mocks
	$(GO) test ./...

test-race: mocks
	$(GO) test -race ./...

cover:
	$(GO) test -cover $(PKGS)

cover-html:
	$(GO) test -coverprofile=coverage.out $(PKGS) && $(GO) tool cover -html=coverage.out -o coverage.html

# Check if current coverage meets the threshold
cover-check:
	@echo "🔍 Checking coverage against threshold..."
	@$(GO) test -coverprofile=coverage.out -covermode=atomic $(PKGS) > /dev/null 2>&1
	@if [ -f ci/coverage_min_percent ]; then \
		MIN=$$(cat ci/coverage_min_percent); \
	else \
		MIN=30; \
	fi; \
	$(GO) tool cover -func=coverage.out | awk -v min="$$MIN" '/^total:/ { \
		pct = substr($$3, 1, length($$3)-1); \
		print "📊 Coverage: " pct "% (minimum: " min "%)"; \
		if (pct < min) { \
			print "❌ Coverage below threshold"; \
			exit 1; \
		} \
		print "✅ Coverage check passed"; \
	}'

# Update the coverage threshold to current coverage (use with caution!)
cover-update-threshold:
	@echo "📈 Updating coverage threshold to current coverage..."
	@$(GO) test -coverprofile=coverage.out -covermode=atomic ./... > /dev/null 2>&1
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {print int(substr($$3, 1, length($$3)-1))}'); \
	echo "$$COVERAGE" > ci/coverage_min_percent; \
	echo "✅ Updated threshold to $$COVERAGE%"

# Show current coverage stats
cover-report:
	@$(GO) test -coverprofile=coverage.out -covermode=atomic $(PKGS) > /dev/null 2>&1
	@echo "📊 Coverage Report:"
	@echo "===================="
	@$(GO) tool cover -func=coverage.out | grep -E "(^github.com/censys/cencli/internal/|^total:)" | \
		awk '{if ($$1 == "total:") print "\n" $$0; else print $$0}' | \
		sort -t: -k3 -rn | head -20
	@echo ""
	@echo "Threshold: $$(cat ci/coverage_min_percent 2>/dev/null || echo "0")%"

docker:
	docker build -t cencli:local .

e2e: $(BINARY)
	$(GO) test -v ./cmd/cencli/e2e

# Run E2E tests with environment variables from .env
# To run a specific fixture, use: make e2e-with-env FIXTURE=view/host-basic
# To run all fixtures for a command, use: make e2e-with-env FIXTURE=view/help
e2e-with-env: $(BINARY)
	@if [ -n "$(FIXTURE)" ]; then \
		echo "Running E2E test for fixture: $(FIXTURE)"; \
		godotenv -f .env bash -c 'CENCLI_ENABLE_E2E_TESTS=true $(GO) test -v ./cmd/cencli/e2e -run TestE2E/$(FIXTURE)'; \
	else \
		godotenv -f .env bash -c 'CENCLI_ENABLE_E2E_TESTS=true $(GO) test -v ./cmd/cencli/e2e'; \
	fi

.PHONY: all verify clean $(BINARY) tools sqlc fmt vet lint test test-race cover cover-html cover-check cover-update-threshold cover-report docker e2e mocks
