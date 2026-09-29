.PHONY: test agents-check rules-check rules-drift-check test-checkout test-installed go-test go-build coverage lint-shell smoke-install clean-test vendor-clean

# Run the CLI test suite (uses BATS_BIN/PATH or the initialized tests/vendor/bats submodule)
test:
	@bash scripts/check-agents-md.sh
	@bash scripts/check-rules-drift.sh
	@GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./rules/tests/check_rules.go
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./cmd/afstest
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./cmd/afsverify

agents-check:
	@bash scripts/check-agents-md.sh

rules-check:
	@GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./rules/tests/check_rules.go

rules-drift-check:
	@bash scripts/check-rules-drift.sh

test-checkout:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./cmd/afstest

test-installed:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./cmd/afsverify

go-test:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go test ./...

go-build:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go build ./cmd/afs

coverage:
	@set -eu; \
	export GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}"; \
	export GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}"; \
	profile="$$(mktemp "$${TMPDIR:-/tmp}/agent47-coverage.XXXXXX")"; \
	trap 'rm -f "$$profile"' EXIT INT TERM; \
	packages="$$(go list ./cmd/afs ./internal/...)"; \
	go test $$packages -coverprofile="$$profile"; \
	total="$$(go tool cover -func="$$profile" | awk '/^total:/ { gsub("%","",$$3); print $$3 }')"; \
	echo "total coverage: $${total}%"; \
	awk -v total="$$total" 'BEGIN { exit !(total+0 >= 80) }'; \
	failed=0; \
	for pkg in $$packages; do \
		output="$$(go test -cover "$$pkg" 2>&1)"; \
		echo "$$output"; \
		coverage="$$(printf '%s\n' "$$output" | awk '/coverage: .* of statements/ { gsub("%","",$$5); print $$5 }' | tail -n 1)"; \
		if [ -z "$$coverage" ]; then continue; fi; \
		awk -v coverage="$$coverage" -v pkg="$$pkg" 'BEGIN { if (coverage+0 < 65) { printf("package coverage below floor: %s (%s%%)\n", pkg, coverage) > "/dev/stderr"; exit 1 } }' || failed=1; \
	done; \
	exit "$$failed"

lint-shell:
	./scripts/lint-shell

smoke-install:
	GOCACHE="$${GOCACHE:-/tmp/agent47-go-build-cache}" GOMODCACHE="$${GOMODCACHE:-/tmp/agent47-go-mod-cache}" go run ./cmd/afssmoke

# Remove any leftover temp dirs from failed/terminated test runs
clean-test:
	find "${TMPDIR:-/tmp}" -maxdepth 1 \( -type d -name 'afs-test-*' -o -type d -name 'afs-skills-*' \) -print -exec rm -rf {} +

# Remove embedded git metadata from vendored deps (e.g., bats)
vendor-clean:
	find tests/vendor -type d -name '.git' -prune -print -exec rm -rf {} +
