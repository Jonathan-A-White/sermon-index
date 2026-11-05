# The sermon index's gate is `make test` and `make lint`.
# The tests start their own Postgres (embedded-postgres): no Docker, no
# installed Postgres, and DATABASE_URL is never read by them. The first run
# downloads the Postgres binaries (about 15 MB) into ~/.cache/sermon-index.

GO ?= go
# The build's version, as /v1/health answers it.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test lint clean

build:
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/sermons ./cmd/sermons

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

clean:
	$(RM) bin/sermons
