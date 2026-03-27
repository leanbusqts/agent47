.PHONY: test test-checkout test-installed go-test go-build lint-shell smoke-install clean-test vendor-clean

# Run the CLI test suite (auto-installs a temporary bats copy from tests/vendor when needed)
test:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go run ./cmd/afstest
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go run ./cmd/afsverify

test-checkout:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go run ./cmd/afstest

test-installed:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go run ./cmd/afsverify

go-test:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go test ./...

go-build:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go build ./cmd/afs

lint-shell:
	./scripts/lint-shell

smoke-install:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" go run ./cmd/afssmoke

# Remove any leftover temp dirs from failed/terminated test runs
clean-test:
	find "${TMPDIR:-/tmp}" -maxdepth 1 \( -type d -name 'afs-test-*' -o -type d -name 'afs-skills-*' \) -print -exec rm -rf {} +

# Remove embedded git metadata from vendored deps (e.g., bats)
vendor-clean:
	find tests/vendor -type d -name '.git' -prune -print -exec rm -rf {} +
