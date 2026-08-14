BINARY     := listicles
INSTALL_DIR := $(HOME)/.local/bin
BUILD_DIR  := bin
RELEASES_DIR := releases
COMMIT     := $(shell git rev-parse HEAD 2>/dev/null || printf dev)

.PHONY: all build build-all install uninstall clean test test-integration test-all

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(BUILD_DIR)/$(BINARY) .
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

build-all:
	@mkdir -p $(RELEASES_DIR)/linux/amd64 $(RELEASES_DIR)/linux/arm64 $(RELEASES_DIR)/darwin/amd64 $(RELEASES_DIR)/darwin/arm64 $(RELEASES_DIR)/windows
	@echo "Building linux/amd64..."
	GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(RELEASES_DIR)/linux/amd64/$(BINARY) .
	@echo "Building linux/arm64..."
	GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(RELEASES_DIR)/linux/arm64/$(BINARY) .
	@echo "Building darwin/amd64..."
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(RELEASES_DIR)/darwin/amd64/$(BINARY) .
	@echo "Building darwin/arm64..."
	GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(RELEASES_DIR)/darwin/arm64/$(BINARY) .
	@echo "Building windows/amd64..."
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(RELEASES_DIR)/windows/$(BINARY).exe .
	@echo "Pre-built binaries written to $(RELEASES_DIR)/"

install:
	@mkdir -p $(INSTALL_DIR)
	@if command -v go >/dev/null 2>&1; then \
		echo "==> Go found - building listicles from source..."; \
		mkdir -p $(BUILD_DIR); \
		go build -ldflags="-s -w -X github.com/wingitman/listicles/internal/version.Commit=$(COMMIT)" -o $(BUILD_DIR)/$(BINARY) . || exit 1; \
		cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY); \
		echo "    Built and installed from source."; \
	else \
		echo "==> Go not found - installing pre-built binary from releases/..."; \
		OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		ARCH=$$(uname -m); \
		case "$$ARCH" in x86_64|amd64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; *) echo "ERROR: Unsupported architecture: $$ARCH"; exit 1 ;; esac; \
		if [ "$$OS" = "darwin" ]; then RELEASE_BIN="$(RELEASES_DIR)/darwin/$$ARCH/$(BINARY)"; elif [ "$$OS" = "linux" ]; then RELEASE_BIN="$(RELEASES_DIR)/linux/$$ARCH/$(BINARY)"; else echo "ERROR: Unsupported OS: $$OS"; exit 1; fi; \
		if [ ! -f "$$RELEASE_BIN" ]; then echo "ERROR: Pre-built binary not found at $$RELEASE_BIN"; echo "       Please install Go (https://go.dev/dl/) and re-run, or ask a developer to run 'make build-all' and commit the releases/ folder."; exit 1; fi; \
		cp "$$RELEASE_BIN" $(INSTALL_DIR)/$(BINARY); \
		chmod +x $(INSTALL_DIR)/$(BINARY); \
		echo "    Installed pre-built binary."; \
	fi
	@"$(INSTALL_DIR)/$(BINARY)" --ensure-config
	@echo "Installed: $(INSTALL_DIR)/$(BINARY)"
	@echo ""
	@$(MAKE) --no-print-directory install-shell

install-shell:
	@# --- zsh ---
	@if [ -f "$(HOME)/.zshrc" ]; then \
		if ! grep -q '\.local/bin' "$(HOME)/.zshrc"; then \
			echo "" >> "$(HOME)/.zshrc"; \
			echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> "$(HOME)/.zshrc"; \
			echo "Added ~/.local/bin to PATH in ~/.zshrc"; \
		fi; \
		if ! grep -q "listicles shell integration" "$(HOME)/.zshrc"; then \
			echo "" >> "$(HOME)/.zshrc"; \
			echo "# listicles shell integration" >> "$(HOME)/.zshrc"; \
			echo "source $(CURDIR)/shell/listicles.zsh" >> "$(HOME)/.zshrc"; \
			echo "Added listicles to ~/.zshrc"; \
		else \
			echo "~/.zshrc already has listicles integration"; \
		fi \
	fi
	@# --- bash ---
	@if [ -f "$(HOME)/.bashrc" ]; then \
		if ! grep -q '\.local/bin' "$(HOME)/.bashrc"; then \
			echo "" >> "$(HOME)/.bashrc"; \
			echo 'export PATH="$$HOME/.local/bin:$$PATH"' >> "$(HOME)/.bashrc"; \
			echo "Added ~/.local/bin to PATH in ~/.bashrc"; \
		fi; \
		if ! grep -q "listicles shell integration" "$(HOME)/.bashrc"; then \
			echo "" >> "$(HOME)/.bashrc"; \
			echo "# listicles shell integration" >> "$(HOME)/.bashrc"; \
			echo "source $(CURDIR)/shell/listicles.bash" >> "$(HOME)/.bashrc"; \
			echo "Added listicles to ~/.bashrc"; \
		else \
			echo "~/.bashrc already has listicles integration"; \
		fi \
	fi
	@# --- fish ---
	@if [ -f "$(HOME)/.config/fish/config.fish" ]; then \
		if ! grep -q '\.local/bin' "$(HOME)/.config/fish/config.fish"; then \
			echo "" >> "$(HOME)/.config/fish/config.fish"; \
			echo "fish_add_path \$$HOME/.local/bin" >> "$(HOME)/.config/fish/config.fish"; \
			echo "Added ~/.local/bin to PATH in ~/.config/fish/config.fish"; \
		fi; \
		if ! grep -q "listicles shell integration" "$(HOME)/.config/fish/config.fish"; then \
			echo "" >> "$(HOME)/.config/fish/config.fish"; \
			echo "# listicles shell integration" >> "$(HOME)/.config/fish/config.fish"; \
			echo "source $(CURDIR)/shell/listicles.fish" >> "$(HOME)/.config/fish/config.fish"; \
			echo "Added listicles to ~/.config/fish/config.fish"; \
		else \
			echo "~/.config/fish/config.fish already has listicles integration"; \
		fi \
	fi
	@# --- powershell ---
	@if [ -f "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1" ]; then \
		if ! grep -q '\.local/bin' "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; then \
			echo "" >> "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; \
			echo '$$env:PATH = "$$HOME\.local\bin;$$env:PATH"' >> "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; \
			echo "Added ~/.local/bin to PATH in PowerShell profile"; \
		fi; \
		if ! grep -q "listicles shell integration" "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; then \
			echo "" >> "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; \
			echo "# listicles shell integration" >> "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; \
			echo ". $(CURDIR)/shell/listicles.ps1" >> "$(HOME)/.config/powershell/Microsoft.PowerShell_profile.ps1"; \
			echo "Added listicles to PowerShell profile"; \
		else \
			echo "PowerShell profile already has listicles integration"; \
		fi \
	fi
	@echo ""
	@echo "Reload your shell or run: source ~/.zshrc  (or ~/.bashrc / fish / . \$$PROFILE for pwsh)"
	@echo "Then type 'l' to launch listicles."

uninstall:
	@rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Removed $(INSTALL_DIR)/$(BINARY)"
	@echo "Note: shell function lines remain in your rc files — remove them manually if desired."

clean:
	rm -rf $(BUILD_DIR)

# Unit tests only (fast, no PTY required, safe for CI)
test:
	go test ./internal/... -timeout 30s

# Integration tests (require a real PTY / display server)
test-integration:
	go test -tags integration -timeout 60s -v .

# Run everything
test-all: test test-integration
