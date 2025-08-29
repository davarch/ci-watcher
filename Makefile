# ---- Settings ---------------------------------------------------------------
APP        := ci-watcher
CMD        := ./cmd/ci-watcher
BIN_DIR    := bin
BIN        := $(BIN_DIR)/$(APP)

# Version from git; falls back to 'dev'
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -s -w -X main.version=$(VERSION)
GOFLAGS    := -trimpath
TESTFLAGS  := -race -count=1

# Paths
LOCAL_BIN  := $(HOME)/.local/bin
CONF_DIR   := $(HOME)/.config/ci-watcher
CONF_FILE  := $(CONF_DIR)/config.yaml
SYSTEMD_USER_DIR := $(HOME)/.config/systemd/user
UNIT_FILE  := $(SYSTEMD_USER_DIR)/$(APP).service

# ---- Helpers ----------------------------------------------------------------
# Print help: make help
.PHONY: help
help:
	@echo "Targets:"
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z0-9_\/-]+:.*##/{printf "  \033[36m%-24s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---- Build / Install --------------------------------------------------------
$(BIN): ## Build release binary
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BIN) $(CMD)

.PHONY: build
build: $(BIN) ## Build release binary

.PHONY: debug
debug: ## Build debug binary (no -s -w)
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -o $(BIN) $(CMD)

.PHONY: install
install: build ## Install binary to ~/.local/bin
	@install -Dm755 $(BIN) $(LOCAL_BIN)/$(APP)
	@echo "Installed -> $(LOCAL_BIN)/$(APP)"

.PHONY: uninstall
uninstall: ## Remove binary from ~/.local/bin
	@rm -f $(LOCAL_BIN)/$(APP)
	@echo "Removed $(LOCAL_BIN)/$(APP)"

# ---- Dev hygiene ------------------------------------------------------------
.PHONY: fmt
fmt: ## go fmt
	go fmt ./...

.PHONY: vet
vet: ## go vet
	go vet ./...

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

# ---- Tests / Coverage -------------------------------------------------------
.PHONY: test
test: ## Run unit tests
	go test $(TESTFLAGS) ./...

.PHONY: cover
cover: ## Run tests with coverage (HTML report at coverage.html)
	go test $(TESTFLAGS) -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html"

# ---- Lint (optional) --------------------------------------------------------
.PHONY: lint
lint: ## Run golangci-lint if present
	@if command -v golangci-lint >/dev/null 2>&1; then \
	  golangci-lint run ./... ; \
	else \
	  echo "golangci-lint not found. Install: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCAL_BIN)"; \
	fi

# ---- Run --------------------------------------------------------------------
.PHONY: run
run: build ## Run scheduler with --config $(CONF_FILE)
	@if [ ! -f "$(CONF_FILE)" ]; then echo "Config not found: $(CONF_FILE)"; exit 2; fi
	$(BIN) run --config $(CONF_FILE)

# ---- Completions ------------------------------------------------------------
COMP_DIR_BASH := $(HOME)/.local/share/bash-completion/completions
COMP_DIR_ZSH  := $(HOME)/.local/share/zsh/site-functions

.PHONY: completion
completion: build ## Install shell completion (bash & zsh)
	@mkdir -p $(COMP_DIR_BASH) $(COMP_DIR_ZSH)
	@$(BIN) completion bash > $(COMP_DIR_BASH)/$(APP)
	@$(BIN) completion zsh  > $(COMP_DIR_ZSH)/_\$(APP)
	@echo "Completion installed for bash and zsh."

# ---- systemd --user ---------------------------------------------------------
define UNIT_CONTENT
[Unit]
Description=CI Watcher (GitLab polling -> notify + waybar cache)
After=network-online.target

[Service]
ExecStart=$(LOCAL_BIN)/$(APP) run --config $(CONF_FILE)
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
endef
export UNIT_CONTENT

.PHONY: systemd-install
systemd-install: install ## Install systemd --user unit
	@mkdir -p $(SYSTEMD_USER_DIR) $(CONF_DIR)
	@if [ ! -f "$(CONF_FILE)" ]; then \
	  if [ -f "./config.yaml" ]; then install -Dm600 ./config.yaml $(CONF_FILE); \
	  elif [ -f "./config.example.yaml" ]; then install -Dm600 ./config.example.yaml $(CONF_FILE); \
	  else touch $(CONF_FILE); fi; \
	  echo "Config -> $(CONF_FILE)"; \
	fi
	@echo "$$UNIT_CONTENT" > $(UNIT_FILE)
	@systemctl --user daemon-reload
	@echo "Unit -> $(UNIT_FILE)"

.PHONY: systemd-start
systemd-start: ## Enable & start service
	systemctl --user enable --now $(APP).service
	systemctl --user status $(APP).service --no-pager

.PHONY: systemd-stop
systemd-stop: ## Stop service
	systemctl --user stop $(APP).service

.PHONY: systemd-disable
systemd-disable: ## Disable service
	systemctl --user disable $(APP).service

.PHONY: logs
logs: ## Follow service logs
	journalctl --user -u $(APP).service -f

# ---- Clean ------------------------------------------------------------------
.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html
