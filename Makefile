.PHONY: help build release snapshot clean install-goreleaser

# Default target
help:
	@echo "Available targets:"
	@echo "  build              - Build binaries for all platforms using goreleaser"
	@echo "  release            - Create a new release (requires a git tag)"
	@echo "  snapshot           - Create a snapshot release (no tag required)"
	@echo "  clean              - Remove build artifacts"
	@echo "  install-goreleaser - Install goreleaser (macOS/Linux)"
	@echo ""
	@echo "Example workflows:"
	@echo "  make snapshot      - Test the release process locally"
	@echo "  make release       - Create and publish a release (run after creating a git tag)"

# Build binaries using goreleaser (snapshot mode)
build: snapshot

# Create a snapshot release (for testing, no git tag required)
snapshot:
	@echo "Creating snapshot release..."
	goreleaser release --snapshot --clean

# Create a full release (requires a git tag)
release:
	@echo "Creating release..."
	@if [ -z "$$(git describe --exact-match --tags 2>/dev/null)" ]; then \
		echo "Error: No git tag found. Create a tag first with: git tag v1.0.0"; \
		exit 1; \
	fi
	goreleaser release --clean

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf dist/
	rm -f iphone2sbr

# Install goreleaser (macOS with Homebrew or Linux)
install-goreleaser:
	@echo "Installing goreleaser..."
	@if command -v brew >/dev/null 2>&1; then \
		echo "Installing via Homebrew..."; \
		brew install goreleaser; \
	else \
		echo "Homebrew not found. Installing via go install..."; \
		go install github.com/goreleaser/goreleaser/v2@latest; \
	fi
