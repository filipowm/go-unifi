# Makefile for go-unifi — local development helpers.
# Not wired into CI (yet); meant for use on your own machine.

# Use bash so recipes behave predictably.
SHELL := bash

GO            ?= go
GOLANGCI_LINT ?= golangci-lint
PKG           := ./...
COVERAGE_RAW  := coverage-all.out
COVERAGE_OUT  := coverage.out
COVERAGE_HTML := coverage.html

# Pass extra flags, e.g. `make test RUN=TestNewClient` or `make test ARGS="-v"`.
RUN  ?=
ARGS ?=
RUN_FLAG := $(if $(RUN),-run $(RUN),)

# Controller version for codegen; `latest` resolves the newest release.
VERSION ?= latest

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

##@ Build & Test

.PHONY: build
build: ## Compile all packages.
	$(GO) build $(PKG)

.PHONY: test
test: ## Run tests with coverage (generated files excluded from the report).
	$(GO) test $(RUN_FLAG) -coverprofile=$(COVERAGE_RAW) -covermode atomic $(ARGS) $(PKG)
	@grep -v '\.generated\.go:' $(COVERAGE_RAW) > $(COVERAGE_OUT)
	$(GO) tool cover -func=$(COVERAGE_OUT) | tail -n 1

.PHONY: test-fast
test-fast: ## Run tests without coverage (quick feedback loop).
	$(GO) test $(RUN_FLAG) $(ARGS) $(PKG)

.PHONY: cover
cover: test ## Run tests and open the HTML coverage report.
	$(GO) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report: $(COVERAGE_HTML)"

##@ Code Quality

.PHONY: lint
lint: ## Run golangci-lint.
	$(GOLANGCI_LINT) run

.PHONY: fmt
fmt: ## Format code via golangci-lint (gofumpt/goimports/gci).
	$(GOLANGCI_LINT) fmt

.PHONY: tidy
tidy: ## Tidy and verify go.mod / go.sum.
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: check
check: build lint test ## Build, lint and test — the pre-push gate.

##@ Code Generation

.PHONY: generate
generate: generate-stringer generate-resources ## Regenerate everything (stringer + resources).

.PHONY: generate-stringer
generate-stringer: ## Regenerate DeviceState stringer.
	$(GO) generate unifi/device.go

.PHONY: generate-resources
generate-resources: ## Regenerate resource types (VERSION=latest|9.3.45|...; downloads the controller).
	$(GO) run ./codegen/ -version-base-dir=./codegen/ -output-dir=./unifi $(VERSION)

##@ Housekeeping

.PHONY: clean
clean: ## Remove build/coverage artifacts.
	$(GO) clean
	rm -f $(COVERAGE_RAW) $(COVERAGE_OUT) $(COVERAGE_HTML)
