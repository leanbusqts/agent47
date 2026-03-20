.PHONY: test lint-shell smoke-install clean-test vendor-clean

# Run the CLI test suite (auto-installs a temporary bats copy from tests/vendor when needed)
test:
	./scripts/test

lint-shell:
	./scripts/lint-shell

smoke-install:
	./scripts/smoke-install

# Remove any leftover temp dirs from failed/terminated test runs
clean-test:
	find "${TMPDIR:-/tmp}" -maxdepth 1 \( -type d -name 'afs-test-*' -o -type d -name 'afs-skills-*' \) -print -exec rm -rf {} +

# Remove embedded git metadata from vendored deps (e.g., bats)
vendor-clean:
	find tests/vendor -type d -name '.git' -prune -print -exec rm -rf {} +
