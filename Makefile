BINARY  := cine
PKG     := github.com/Divyesh172/cine
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \\
	-X '$(PKG)/cmd.Version=$(VERSION)' \\
	-X '$(PKG)/cmd.Commit=$(COMMIT)' \\
	-X '$(PKG)/cmd.Date=$(DATE)'

.PHONY: all build test lint fmt vet tidy clean install

all: fmt vet test build

# Reproducible, static binary. modernc.org/sqlite is pure Go, so CGO stays off.
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./... -race -count=1

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -s -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || true

tidy:
	go mod tidy

install: build
	install -d $(HOME)/.local/bin
	install -m 0755 $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
	go clean
