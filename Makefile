BINARY  := sourceseedy
PKG     := github.com/drewstinnett/sourceseedy/cli/cmd
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GOBIN   ?= $(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)

LDFLAGS := -s -w \
	-X $(PKG).version=$(VERSION) \
	-X $(PKG).commit=$(COMMIT) \
	-X $(PKG).date=$(DATE)

.PHONY: build install
build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o dist/$(BINARY) ./cli/

install:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(GOBIN)/$(BINARY) ./cli/
