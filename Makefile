# The sermon index's gate is `make test` and `make lint`.
# The tests start their own Postgres (embedded-postgres): no Docker, no
# installed Postgres, and DATABASE_URL is never read by them. The first run
# downloads the Postgres binaries (about 15 MB) into ~/.cache/sermon-index.

GO ?= go
# The build's version, as /v1/health answers it.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: build test lint clean install-laptop smoke henry-leaves

build:
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/sermons ./cmd/sermons

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

install-laptop:
	sh scripts/install-laptop.sh

smoke:
	sh scripts/smoke.sh

# Fetch the Archive's OCR of the two Matthew Henry scans (about 150 MB, into the
# git-ignored .archive-cache/) and check that every henry-sermons url opens the
# leaf that prints its sermon's head. `make test` runs the check when the OCR is
# already cached and skips it when it is not.
henry-leaves:
	sh scripts/fetch-henry-ocr.sh
	$(GO) test ./internal/sources -run 'TestHenryLeaf' -count=1 -v

clean:
	$(RM) bin/sermons
