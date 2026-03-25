BINARY     := listicles
INSTALL_DIR := $(HOME)/.local/bin
BUILD_DIR  := bin

.PHONY: all build install uninstall clean test test-integration test-all

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) .
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

install: build
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed: $(INSTALL_DIR)/$(BINARY)"
	@echo ""
	@$(MAKE) --no-print-directory install-shell

install-shell:
	@# --- zsh ---
	@if [ -f "$(HOME)/.zshrc" ]; then \
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
