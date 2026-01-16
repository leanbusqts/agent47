.PHONY: test clean-test vendor-clean

# Run the CLI test suite (requires bats-core, vendored or on PATH)
test:
	./scripts/test

# Remove any leftover temp dirs from failed/terminated test runs
clean-test:
	find /tmp -maxdepth 1 -type d -name 'a47-test-*' -print -exec rm -rf {} +

# Remove embedded git metadata from vendored deps (e.g., bats)
vendor-clean:
	find tests/vendor -type d -name '.git' -prune -print -exec rm -rf {} +
