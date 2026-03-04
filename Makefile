.PHONY: build run gateway test setup clean docker-up docker-down lint prod start stop restart install-ffmpeg

BINARY    := aevitas
BUILD_DIR := .
CONFIG    := $(HOME)/.aevitas/config.json
INSTALL_DIR := $(HOME)/.aevitas/bin
SCRIPT_DIR := scripts

## Build
build:
	go build -o $(BINARY) ./cmd/aevitas

## Run agent REPL
run: build
	./$(BINARY) agent

## Run gateway (channels + cron + heartbeat)
gateway: build
	./$(BINARY) gateway

## Run onboard to initialize config and workspace
## Initialize/reset workspace files
onboard: build
	./$(BINARY) onboard

## Show status
status: build
	./$(BINARY) status

## List installed skills
skills-list: build
	./$(BINARY) skills list

## Install or update skills (usage: make skills-install [skill-name])
skills-install: build
	@SKILL="$(filter-out $@,$(MAKECMDGOALS))"; \
	if [ -z "$$SKILL" ]; then \
		read -p "Install all skills? [y/N] " -n 1 -r; echo; \
		if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
			./$(BINARY) skills install; \
		fi \
	else \
		read -p "Install skill '$$SKILL'? [y/N] " -n 1 -r; echo; \
		if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
			./$(BINARY) skills install $$SKILL; \
		fi \
	fi

## Update skills (usage: make skills-update [skill-name])
skills-update: build
	@SKILL="$(filter-out $@,$(MAKECMDGOALS))"; \
	if [ -z "$$SKILL" ]; then \
		read -p "Update all skills? [y/N] " -n 1 -r; echo; \
		if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
			./$(BINARY) skills update; \
		fi \
	else \
		read -p "Update skill '$$SKILL'? [y/N] " -n 1 -r; echo; \
		if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
			./$(BINARY) skills update $$SKILL; \
		fi \
	fi

## Uninstall a skill (usage: make skills-uninstall <skill-name>)
skills-uninstall: build
	@SKILL="$(filter-out $@,$(MAKECMDGOALS))"; \
	if [ -z "$$SKILL" ]; then \
		echo "Usage: make skills-uninstall <skill-name>"; \
		exit 1; \
	fi; \
	read -p "Uninstall skill '$$SKILL'? [y/N] " -n 1 -r; echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		./$(BINARY) skills uninstall $$SKILL; \
	fi

# Prevent make from treating skill names as targets
%:
	@:

## Verify skills integrity
skills-verify: build
	./$(BINARY) skills verify

## Build and install to production directory
prod: install-ffmpeg
	@echo "Tidying dependencies..."
	@go mod tidy
	@$(MAKE) build
	@echo "Installing aevitas to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "✓ aevitas installed to $(INSTALL_DIR)/$(BINARY)"
	@echo "✓ ffmpeg tools installed to $(INSTALL_DIR)/ffmpeg and $(INSTALL_DIR)/ffprobe"
	@echo "Use 'make start' or 'scripts/start.sh' to start in background"

## Download local ffmpeg/ffprobe binaries for production
install-ffmpeg:
	@echo "Installing local ffmpeg/ffprobe into $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	@ARCH="$$(uname -m)"; \
	OS="$$(uname -s)"; \
	if [ "$$OS" != "Darwin" ]; then \
		echo "⚠️ Auto ffmpeg download currently supports macOS only (detected $$OS)."; \
		echo "Please place ffmpeg and ffprobe in $(INSTALL_DIR) manually."; \
		exit 1; \
	fi; \
	case "$$ARCH" in \
		arm64) \
			FFMPEG_URL="https://ffmpeg.martin-riedl.de/download/macos/arm64/1712343170_7.0/ffmpeg.zip"; \
			FFPROBE_URL="https://ffmpeg.martin-riedl.de/download/macos/arm64/1712343170_7.0/ffprobe.zip"; \
			;; \
		x86_64) \
			FFMPEG_URL="https://evermeet.cx/ffmpeg/getrelease/zip"; \
			FFPROBE_URL="https://evermeet.cx/ffmpeg/getrelease/ffprobe/zip"; \
			;; \
		*) \
			echo "❌ Unsupported macOS architecture: $$ARCH"; \
			exit 1; \
			;; \
	esac; \
	TMP_DIR="$$(mktemp -d)"; \
	trap 'rm -rf "$$TMP_DIR"' EXIT; \
	echo "Downloading ffmpeg from $$FFMPEG_URL"; \
	curl -fL "$$FFMPEG_URL" -o "$$TMP_DIR/ffmpeg.zip"; \
	echo "Downloading ffprobe from $$FFPROBE_URL"; \
	curl -fL "$$FFPROBE_URL" -o "$$TMP_DIR/ffprobe.zip"; \
	unzip -qo "$$TMP_DIR/ffmpeg.zip" -d "$$TMP_DIR/ffmpeg"; \
	unzip -qo "$$TMP_DIR/ffprobe.zip" -d "$$TMP_DIR/ffprobe"; \
	if [ ! -f "$$TMP_DIR/ffmpeg/ffmpeg" ] || [ ! -f "$$TMP_DIR/ffprobe/ffprobe" ]; then \
		echo "❌ Downloaded archive missing ffmpeg/ffprobe binaries"; \
		exit 1; \
	fi; \
	cp "$$TMP_DIR/ffmpeg/ffmpeg" "$(INSTALL_DIR)/ffmpeg"; \
	cp "$$TMP_DIR/ffprobe/ffprobe" "$(INSTALL_DIR)/ffprobe"; \
	chmod +x "$(INSTALL_DIR)/ffmpeg" "$(INSTALL_DIR)/ffprobe"; \
	echo "✓ ffmpeg + ffprobe installed to $(INSTALL_DIR)"

## Start gateway in background (production mode)
start:
	@bash $(SCRIPT_DIR)/start.sh

## Stop gateway gracefully
stop:
	@bash $(SCRIPT_DIR)/stop.sh

## Restart gateway
restart:
	@bash $(SCRIPT_DIR)/restart.sh

## Interactive setup: generate config.json
setup:
	@bash scripts/setup.sh

## Run all tests
test:
	go test ./... -count=1

## Run tests with race detection
test-race:
	go test -race ./... -count=1

## Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## Docker: build and start
docker-up:
	docker compose up -d --build

## Docker: stop
docker-down:
	docker compose down

## Clean build artifacts
clean:
	rm -f $(BINARY) coverage.out

## Lint (requires golangci-lint)
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install: brew install golangci-lint"; exit 1; }
	golangci-lint run ./...

## Help
help:
	@echo "aevitas Makefile targets:"
	@echo ""
	@echo "Core:"
	@echo "  build            Build binary"
	@echo "  run              Run agent REPL"
	@echo "  gateway          Start gateway (channels + cron + heartbeat)"
	@echo "  onboard          Initialize config and workspace"
	@echo "  status           Show aevitas status"
	@echo "  setup            Interactive config setup"
	@echo "  prod             Build + install + download ffmpeg/ffprobe"
	@echo ""
	@echo "Production Control:"
	@echo "  start            Start gateway in background"
	@echo "  stop             Stop gateway gracefully"
	@echo "  restart          Restart gateway"
	@echo ""
	@echo "Skills Management:"
	@echo "  skills-list         List installed skills"
	@echo "  skills-install [name] Install skill(s) (name or all)"
	@echo "  skills-update [name]  Update skill(s) (name or all)"
	@echo "  skills-uninstall <name> Uninstall a skill (required)"
	@echo "  skills-verify       Verify skills integrity"
	@echo ""
	@echo "Testing:"
	@echo "  test             Run all tests"
	@echo "  test-race        Run tests with race detection"
	@echo "  test-cover       Run tests with coverage report"
	@echo ""
	@echo "Deployment:"
	@echo "  docker-up        Docker build and start"
	@echo "  docker-down      Docker stop"
	@echo ""
	@echo "Production Scripts (or use make commands above):"
	@echo "  ./scripts/start.sh   Start gateway in background"
	@echo "  ./scripts/stop.sh    Stop gateway gracefully"
	@echo "  ./scripts/restart.sh Restart gateway"
	@echo ""
	@echo "Utilities:"
	@echo "  clean            Remove build artifacts"
	@echo "  lint             Run golangci-lint"
