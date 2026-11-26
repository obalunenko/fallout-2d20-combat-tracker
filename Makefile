APP_NAME := fallout-tracker
CMD_PATH := ./cmd/fallout-tracker
BIN_DIR := ./bin
MIGRATIONS_DIR := ./internal/store/sqlite/migrations
SCHEMA_SQL := ./internal/store/sqlite/sqlc/schema.sql
DB_DOC_DIR ?= ./docs/db
DB ?= ./tracker.db
TOOLS_BIN_DIR := ./tools/bin
TOOLS_GOBIN := $(abspath $(TOOLS_BIN_DIR))
GOOSE_TOOL_DIR := ./tools/goose
GOOSE_BIN := $(TOOLS_BIN_DIR)/goose
SQLC_TOOL_DIR := ./tools/sqlc
SQLC_BIN := $(TOOLS_BIN_DIR)/sqlc
TBLS_TOOL_DIR := ./tools/tbls
TBLS_BIN := $(TOOLS_BIN_DIR)/tbls
GOLANGCI_TOOL_DIR := ./tools/golangci-lint
GOLANGCI_BIN := $(TOOLS_BIN_DIR)/golangci-lint
GORELEASER_TOOL_DIR := ./tools/goreleaser
GORELEASER_BIN := $(TOOLS_BIN_DIR)/goreleaser

# Spec Kit pins. agent-context ships with the Specify CLI, so SPECKIT_VERSION
# pins it too.
SPECKIT_VERSION := 1.2.5
SPECKIT_COMPANION_VERSION := 0.25.0
SPECKIT_BROWNFIELD_VERSION := 1.0.0
SPECKIT_BUGFIX_VERSION := 1.1.0
SPECKIT_FEATURE_NUMBERING_VERSION := 1.0.0
SPECKIT_TOOL_DIR := ./tools/uv
SPECIFY_BIN := $(TOOLS_BIN_DIR)/specify
SPECKIT_ENV := \
	SPECKIT_VERSION='$(SPECKIT_VERSION)' \
	SPECKIT_COMPANION_VERSION='$(SPECKIT_COMPANION_VERSION)' \
	SPECKIT_BROWNFIELD_VERSION='$(SPECKIT_BROWNFIELD_VERSION)' \
	SPECKIT_BUGFIX_VERSION='$(SPECKIT_BUGFIX_VERSION)' \
	SPECKIT_FEATURE_NUMBERING_VERSION='$(SPECKIT_FEATURE_NUMBERING_VERSION)' \
	UV_TOOL_DIR='$(abspath $(SPECKIT_TOOL_DIR))' \
	UV_TOOL_BIN_DIR='$(abspath $(TOOLS_BIN_DIR))' \
	SPECIFY_BIN='$(abspath $(SPECIFY_BIN))'

.PHONY: help run test build fmt tidy tools-list tools-tidy tools-verify goose-install sqlc-install tbls-install schema-generate sqlc-generate db-doc-generate db-check goose-status goose-create vet lint lint-install goreleaser-install goreleaser-check goreleaser-local goreleaser-snapshot ci-check speckit-install speckit-skills-normalize speckit-update-check speckit-update-test clean

help:
	@echo "Targets:"
	@echo "  make run    - Run desktop app"
	@echo "  make test   - Run tests"
	@echo "  make build  - Build binary to ./bin"
	@echo "  make fmt    - Format Go code"
	@echo "  make tidy   - Tidy go.mod/go.sum"
	@echo "  make tools-list - List tools from all tool modules"
	@echo "  make tools-tidy - Tidy go.mod/go.sum of all tool modules"
	@echo "  make tools-verify - Verify tool dependency integrity for all tool modules"
	@echo "  make schema-generate - Rebuild sqlc/schema.sql from a clean migrated DB"
	@echo "  make sqlc-generate - Generate typed DB code via sqlc"
	@echo "  make db-doc-generate - Generate DB docs via tbls from a clean migrated DB"
	@echo "  make db-check - Regenerate sqlc code and run tests"
	@echo "  make vet    - Run go vet"
	@echo "  make lint   - Run golangci-lint v2"
	@echo "  make goreleaser-check - Validate .goreleaser.yaml"
	@echo "  make goreleaser-local - Build binary for current OS/ARCH via GoReleaser"
	@echo "  make goreleaser-snapshot - Build release artifacts locally (no publish)"
	@echo "  make ci-check - Run vet, lint, tests, build and Spec Kit checker tests"
	@echo "  make goose-status                   - Goose status via go mod tool"
	@echo "  make goose-create NAME=add_table    - Create SQL migration"
	@echo "  make speckit-install - Install pinned Spec Kit, extensions and Claude Code skills"
	@echo "  make speckit-skills-normalize - Rewrite Spec Kit skill invocations to /speckit-name"
	@echo "  make speckit-update-check - Check Spec Kit updates without installing them"
	@echo "  make speckit-update-test - Run the Spec Kit update-checker tests"
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
	go -C $(GOOSE_TOOL_DIR) list tool
	go -C $(SQLC_TOOL_DIR) list tool
	go -C $(TBLS_TOOL_DIR) list tool
	go -C $(GOLANGCI_TOOL_DIR) list tool
	go -C $(GORELEASER_TOOL_DIR) list tool

tools-tidy:
	go -C $(GOOSE_TOOL_DIR) mod tidy
	go -C $(SQLC_TOOL_DIR) mod tidy
	go -C $(TBLS_TOOL_DIR) mod tidy
	go -C $(GOLANGCI_TOOL_DIR) mod tidy
	go -C $(GORELEASER_TOOL_DIR) mod tidy

tools-verify:
	go -C $(GOOSE_TOOL_DIR) mod verify
	go -C $(SQLC_TOOL_DIR) mod verify
	go -C $(TBLS_TOOL_DIR) mod verify
	go -C $(GOLANGCI_TOOL_DIR) mod verify
	go -C $(GORELEASER_TOOL_DIR) mod verify

goose-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_GOBIN) go -C $(GOOSE_TOOL_DIR) install github.com/pressly/goose/v3/cmd/goose

sqlc-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_GOBIN) go -C $(SQLC_TOOL_DIR) install github.com/sqlc-dev/sqlc/cmd/sqlc

tbls-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_GOBIN) go -C $(TBLS_TOOL_DIR) install github.com/k1LoW/tbls

schema-generate:
	go run ./internal/store/sqlite/cmd/genschema -migrations $(MIGRATIONS_DIR) -out $(SCHEMA_SQL)

sqlc-generate: schema-generate sqlc-install
	$(SQLC_BIN) generate

db-doc-generate: goose-install tbls-install
	@tmp_dir=$$(mktemp -d); \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	tmp_db="$$tmp_dir/schema.db"; \
	echo "Creating clean SQLite database for docs: $$tmp_db"; \
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) sqlite3 "$$tmp_db" up; \
	echo "Generating database documentation into $(DB_DOC_DIR)"; \
	TBLS_DSN="sqlite://$$tmp_db" TBLS_DOC_PATH="$(DB_DOC_DIR)" $(TBLS_BIN) doc --rm-dist --config .tbls.yml

db-check: sqlc-generate test

vet:
	go vet ./...

lint-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_GOBIN) go -C $(GOLANGCI_TOOL_DIR) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint

lint: lint-install
	$(GOLANGCI_BIN) run ./...

goreleaser-install:
	mkdir -p $(TOOLS_BIN_DIR)
	GOBIN=$(TOOLS_GOBIN) go -C $(GORELEASER_TOOL_DIR) install github.com/goreleaser/goreleaser/v2

goreleaser-check: goreleaser-install
	$(GORELEASER_BIN) check
	$(GORELEASER_BIN) check --config .goreleaser.darwin.yaml

goreleaser-local: goreleaser-install
	mkdir -p $(BIN_DIR)
	$(GORELEASER_BIN) build \
		--clean \
		--snapshot \
		--single-target \
		--id fallout-tracker \
		--output "$(BIN_DIR)/$(APP_NAME)-$$(go env GOOS)-$$(go env GOARCH)$$(go env GOEXE)"

goreleaser-snapshot: goreleaser-install
	$(GORELEASER_BIN) release --snapshot --clean --skip=publish

ci-check: vet lint test build speckit-update-test

goose-status: goose-install
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) sqlite3 $(DB) status

goose-create: goose-install
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) create $(NAME) sql

speckit-install:
	$(SPECKIT_ENV) scripts/install-speckit.sh

speckit-skills-normalize:
	scripts/normalize-speckit-skills.sh

speckit-update-check:
	$(SPECKIT_ENV) scripts/check-speckit-updates.py

speckit-update-test:
	python3 scripts/test_check_speckit_updates.py

clean:
	rm -rf $(BIN_DIR)
