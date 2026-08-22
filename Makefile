# Portable build for Linux and OpenBSD. Works with both GNU make and OpenBSD's
# make, so it avoids GNU-only constructs (no ifeq, no $(shell), no % rules).
#
# On NixOS use the flake instead: nix build / nix develop.
#
# Cross-compiling needs no C toolchain, since the SQLite driver is pure Go:
#     make GOOS=openbsd GOARCH=amd64
# Supported by the driver: linux and openbsd on amd64 and arm64, plus
# linux/arm, freebsd, netbsd/amd64, darwin and windows.

GO          ?= go
GOOS        ?=
GOARCH      ?=

# Not called LDFLAGS: BSD make predefines that one, so ?= would never apply and
# the flags would be silently dropped on OpenBSD. (go reads GOFLAGS from the
# environment by itself, so there is nothing to wire up for it here.)
GO_LDFLAGS  ?= -s -w

# No cgo, so no C compiler is required. This produces a static binary on Linux;
# on OpenBSD the result still links against libc, which the kernel requires.
CGO_ENABLED ?= 0

DESTDIR     ?=
PREFIX      ?= /usr/local
BINDIR      ?= $(PREFIX)/bin
SHAREDIR    ?= $(PREFIX)/share/iosu

# Keep in step with the sqlc in the flake's dev shell. If they diverge, the
# generated code changes and CI's drift check fails.
SQLC_VERSION = v1.31.1
SQLC        ?= $(GO) run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)

BUILD = CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	$(GO) build -ldflags "$(GO_LDFLAGS)"

.PHONY: all iosud iosu test test-race vet fmt check generate install uninstall clean help

all: iosud iosu

iosud:
	$(BUILD) -o iosud ./cmd/iosud

iosu:
	$(BUILD) -o iosu ./cmd/iosu

test:
	$(GO) test ./...

# The race detector needs cgo and is not available on OpenBSD.
test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	@unformatted=`gofmt -l .`; \
	if [ -n "$$unformatted" ]; then \
		echo "these files are not gofmt'd:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

check: fmt vet test

# Regenerate internal/store/sqlc after changing a migration or a query.
generate:
	$(SQLC) generate

install: all
	mkdir -p $(DESTDIR)$(BINDIR)
	install -m 755 iosud $(DESTDIR)$(BINDIR)/iosud
	install -m 755 iosu $(DESTDIR)$(BINDIR)/iosu
	mkdir -p $(DESTDIR)$(SHAREDIR)
	install -m 644 config.example.toml $(DESTDIR)$(SHAREDIR)/config.example.toml

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/iosud
	rm -f $(DESTDIR)$(BINDIR)/iosu
	rm -f $(DESTDIR)$(SHAREDIR)/config.example.toml

clean:
	rm -f iosu iosud

help:
	@echo 'targets:'
	@echo '  all         build iosud and iosu (default)'
	@echo '  test        run the test suite'
	@echo '  test-race   run it under the race detector (not on OpenBSD)'
	@echo '  check       fmt, vet and test'
	@echo '  generate    regenerate internal/store/sqlc with sqlc'
	@echo '  install     install into $$DESTDIR$$PREFIX (default /usr/local)'
	@echo '  clean       remove the built binaries'
	@echo
	@echo 'cross-compile: make GOOS=openbsd GOARCH=amd64'
