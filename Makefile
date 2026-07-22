BINARY    := ncp
BIN_DIR   := bin
MAIN      := ./cmd/ncp
COVER_OUT := coverage.out
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

export CGO_ENABLED := 1

# Colors (honor https://no-color.org)
ifndef NO_COLOR
BOLD  := \033[1m
DIM   := \033[2m
CYAN  := \033[36m
GREEN := \033[32m
RESET := \033[0m
endif

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Show this help
	@printf '\n$(BOLD)$(BINARY)$(RESET) — Namecheap CLI/TUI $(DIM)($(VERSION))$(RESET)\n'
	@awk 'BEGIN {FS = ":.*##"} \
		/^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(CYAN)%-12s$(RESET) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n$(BOLD)%s$(RESET)\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@printf '\n'

##@ Development

.PHONY: build
build: ## Build the ncp binary into bin/
	@printf '$(CYAN)▶$(RESET) building $(BOLD)$(BINARY)$(RESET) $(DIM)($(VERSION))$(RESET)\n'
	@go build -o $(BIN_DIR)/$(BINARY) $(MAIN)
	@printf '$(GREEN)✓$(RESET) $(BIN_DIR)/$(BINARY)\n'

.PHONY: run
run: build ## Build and launch the TUI
	@./$(BIN_DIR)/$(BINARY)

.PHONY: install
install: ## Install ncp into GOPATH/bin
	@printf '$(CYAN)▶$(RESET) installing $(BOLD)$(BINARY)$(RESET)\n'
	@go install $(MAIN)

##@ Quality

.PHONY: fmt
fmt: ## Format all Go files with gofumpt
	@printf '$(CYAN)▶$(RESET) gofumpt\n'
	@gofumpt -l -w .

.PHONY: lint
lint: ## Run golangci-lint
	@printf '$(CYAN)▶$(RESET) golangci-lint\n'
	@golangci-lint run

.PHONY: test
test: ## Run all tests with the race detector
	@printf '$(CYAN)▶$(RESET) go test -race\n'
	@go test -race ./...

.PHONY: cover
cover: ## Run tests with coverage and print the summary
	@printf '$(CYAN)▶$(RESET) go test -cover\n'
	@go test -race -covermode=atomic -coverprofile=$(COVER_OUT) ./...
	@go tool cover -func=$(COVER_OUT) | tail -1

.PHONY: cover-html
cover-html: cover ## Open the HTML coverage report
	@go tool cover -html=$(COVER_OUT)

.PHONY: check
check: fmt lint test ## Everything CI runs: fmt + lint + test

.PHONY: screenshots
screenshots: ## Regenerate the README screenshots in assets/ (needs tmux + freeze)
	@printf '$(CYAN)▶$(RESET) regenerating screenshots\n'
	@zsh scripts/screenshots/regen.sh
	@printf '$(GREEN)✓$(RESET) assets/\n'

##@ Setup

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	@go mod tidy

.PHONY: hooks
hooks: ## Install pre-commit hooks (incl. conventional commit-msg check)
	@pre-commit install --install-hooks
	@pre-commit install --hook-type commit-msg
	@printf '$(GREEN)✓$(RESET) hooks installed\n'

##@ Housekeeping

.PHONY: clean
clean: ## Remove build artifacts and coverage output
	@rm -rf $(BIN_DIR) $(COVER_OUT)
	@printf '$(GREEN)✓$(RESET) cleaned\n'
