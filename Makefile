APP_NAME := fallout-tracker
CMD_PATH := ./cmd/fallout-tracker
BIN_DIR := ./bin
MIGRATIONS_DIR := ./internal/store/sqlite/migrations
SCHEMA_SQL := ./internal/store/sqlite/sqlc/schema.sql
DB ?= ./tracker.db
TOOLS_BIN_DIR := ./tools/bin
GOOSE_TOOL_DIR := ./tools/goose
GOOSE_TOOL_MODFILE := go.tool.mod
GOOSE_BIN := $(TOOLS_BIN_DIR)/goose
SQLC_TOOL_DIR := ./tools/sqlc
SQLC_TOOL_MODFILE := go.tool.mod
SQLC_BIN := $(TOOLS_BIN_DIR)/sqlc
GOLANGCI_TOOL_DIR := ./tools/golangci-lint
GOLANGCI_TOOL_MODFILE := go.tool.mod
GOLANGCI_BIN := $(TOOLS_BIN_DIR)/golangci-lint
GORELEASER_TOOL_DIR := ./tools/goreleaser
GORELEASER_TOOL_MODFILE := go.tool.mod
GORELEASER_BIN := $(TOOLS_BIN_DIR)/goreleaser

.PHONY: help run test build fmt tidy tools-list tools-verify goose-install sqlc-install schema-generate sqlc-generate db-check goose-status goose-create vet lint lint-install goreleaser-install goreleaser-check goreleaser-local goreleaser-snapshot ci-check clean

help:
	@echo "Targets:"
	@echo "  make run    - Run desktop app"
	@echo "  make test   - Run tests"
	@echo "  make build  - Build binary to ./bin"
	@echo "  make fmt    - Format Go code"
	@echo "  make tidy   - Tidy go.mod/go.sum"
	@echo "  make tools-list - List tools from all tool modules"
	@echo "  make tools-verify - Verify tool dependency integrity for all tool modules"
	@echo "  make schema-generate - Rebuild sqlc/schema.sql from a clean migrated DB"
	@echo "  make sqlc-generate - Generate typed DB code via sqlc"
	@echo "  make db-check - Regenerate sqlc code and run tests"
	@echo "  make vet    - Run go vet"
	@echo "  make lint   - Run golangci-lint v2"
	@echo "  make goreleaser-check - Validate .goreleaser.yaml"
	@echo "  make goreleaser-local - Build binary for current OS/ARCH via GoReleaser"
	@echo "  make goreleaser-snapshot - Build release artifacts locally (no publish)"
	@echo "  make ci-check - Run vet, lint, tests and build"
	@echo "  make goose-status                   - Goose status via go mod tool"
	@echo "  make goose-create NAME=add_table    - Create SQL migration"
	@echo "  make clean  - Remove build artifacts"

run: build
	@set +e; \
	trap 'echo "run interrupted by signal (treated as normal stop)"; exit 0' INT TERM; \
	$(BIN_DIR)/$(APP_NAME); \
	status=$$?; \
	if [ $$status -eq 130 ] || [ $$status -eq 143 ]; then \
		echo "run interrupted by signal (treated as normal stop)"; \
		exit 0; \
	fi; \
	exit $$status

test:
	go test ./...

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

fmt:
	gofmt -w $$(go list -f '{{.Dir}}' ./...)

tidy:
	go mod tidy

tools-list:
	go list -modfile=$(GOOSE_TOOL_DIR)/$(GOOSE_TOOL_MODFILE) tool
	go list -modfile=$(SQLC_TOOL_DIR)/$(SQLC_TOOL_MODFILE) tool
	go list -modfile=$(GOLANGCI_TOOL_DIR)/$(GOLANGCI_TOOL_MODFILE) tool
	go list -modfile=$(GORELEASER_TOOL_DIR)/$(GORELEASER_TOOL_MODFILE) tool

tools-verify:
	go mod verify -modfile=$(GOOSE_TOOL_DIR)/$(GOOSE_TOOL_MODFILE)
	go mod verify -modfile=$(SQLC_TOOL_DIR)/$(SQLC_TOOL_MODFILE)
	go mod verify -modfile=$(GOLANGCI_TOOL_DIR)/$(GOLANGCI_TOOL_MODFILE)
	go mod verify -modfile=$(GORELEASER_TOOL_DIR)/$(GORELEASER_TOOL_MODFILE)

goose-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$$(pwd)/$(TOOLS_BIN_DIR) go install -C $(GOOSE_TOOL_DIR) -modfile=$(GOOSE_TOOL_MODFILE) github.com/pressly/goose/v3/cmd/goose

sqlc-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$$(pwd)/$(TOOLS_BIN_DIR) go install -C $(SQLC_TOOL_DIR) -modfile=$(SQLC_TOOL_MODFILE) github.com/sqlc-dev/sqlc/cmd/sqlc

schema-generate:
	go run ./internal/store/sqlite/cmd/genschema -migrations $(MIGRATIONS_DIR) -out $(SCHEMA_SQL)

sqlc-generate: schema-generate sqlc-install
	$(SQLC_BIN) generate

db-check: sqlc-generate test

vet:
	go vet ./...

lint-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$$(pwd)/$(TOOLS_BIN_DIR) go install -C $(GOLANGCI_TOOL_DIR) -modfile=$(GOLANGCI_TOOL_MODFILE) github.com/golangci/golangci-lint/v2/cmd/golangci-lint

lint: lint-install
	$(GOLANGCI_BIN) run ./...

goreleaser-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$$(pwd)/$(TOOLS_BIN_DIR) go install -C $(GORELEASER_TOOL_DIR) -modfile=$(GORELEASER_TOOL_MODFILE) github.com/goreleaser/goreleaser/v2

goreleaser-check:
	go tool -modfile=$(GORELEASER_TOOL_DIR)/$(GORELEASER_TOOL_MODFILE) goreleaser check
	go tool -modfile=$(GORELEASER_TOOL_DIR)/$(GORELEASER_TOOL_MODFILE) goreleaser check --config .goreleaser.darwin.yaml

goreleaser-local:
	mkdir -p $(BIN_DIR)
	go tool -modfile=$(GORELEASER_TOOL_DIR)/$(GORELEASER_TOOL_MODFILE) goreleaser build \
		--clean \
		--snapshot \
		--single-target \
		--id fallout-tracker \
		--output "$(BIN_DIR)/$(APP_NAME)-$$(go env GOOS)-$$(go env GOARCH)$$(go env GOEXE)"

goreleaser-snapshot:
	go tool -modfile=$(GORELEASER_TOOL_DIR)/$(GORELEASER_TOOL_MODFILE) goreleaser release --snapshot --clean --skip=publish

ci-check: vet lint test build

goose-status: goose-install
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) sqlite3 $(DB) status

goose-create: goose-install
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) create $(NAME) sql

clean:
	rm -rf $(BIN_DIR)
