# The sermon index's gate is `make test` and `make lint`.
# The tests start their own Postgres (embedded-postgres): no Docker, no
# installed Postgres, and DATABASE_URL is never read by them. The first run
# downloads the Postgres binaries (about 15 MB) into ~/.cache/sermon-index.

GO ?= go

.PHONY: build test lint clean

build:
	$(GO) build -o bin/sermons ./cmd/sermons

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

clean:
	$(RM) bin/sermons
