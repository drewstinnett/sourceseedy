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

SHELL_SCRIPTS := install.sh .github/scripts/install_test.sh

.PHONY: build install lint-sh hooks
build:
	go build -trimpath -ldflags '$(LDFLAGS)' -o dist/$(BINARY) ./cli/

install:
	go build -trimpath -ldflags '$(LDFLAGS)' -o $(GOBIN)/$(BINARY) ./cli/

# The same check CI runs on the shell scripts
lint-sh:
	@command -v shellcheck >/dev/null || { echo "shellcheck isn't installed (brew install shellcheck)"; exit 1; }
	shellcheck $(SHELL_SCRIPTS)

# Turn on the git hooks in .githooks, which run lint-sh before a push
hooks:
	git config core.hooksPath .githooks
