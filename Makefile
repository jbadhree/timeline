# Local and CI checks. `make check` is the one command to run before pushing.

GOLANGCI_LINT_VERSION := v2.14.0
BIN := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

.PHONY: check fmt-check build lint test race plan release

check: fmt-check build lint test

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

build:
	CGO_ENABLED=0 go build ./...

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

test:
	go test ./...

race:
	go test -race ./...

# Validate this repository's own goals, tasks, and decisions without writing.
# Not part of `check`: drafts are checked against the current time.
plan:
	go run ./cmd/tline check --dry-run

$(GOLANGCI_LINT):
	mkdir -p $(BIN)
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(BIN)/golangci-lint $@

# Build release archives for every supported platform into dist/, with a
# SHA256SUMS file. VERSION must be a tag such as v0.1.0.
VERSION ?=
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
LDFLAGS := -s -w -X github.com/jbadhree/timeline/internal/cli.Version=$(VERSION)

release:
	@case "$(VERSION)" in v[0-9]*.[0-9]*.[0-9]*) ;; *) echo "set VERSION, such as make release VERSION=v0.1.0"; exit 1;; esac
	rm -rf dist && mkdir -p dist
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; name=tline_$(VERSION)_$${os}_$${arch}; \
		echo "building $$name"; \
		mkdir -p dist/$$name && \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o dist/$$name/tline ./cmd/tline && \
		cp README.md dist/$$name/ && \
		tar -C dist -czf dist/$$name.tar.gz $$name && rm -rf dist/$$name || exit 1; \
	done
	cd dist && shasum -a 256 *.tar.gz > SHA256SUMS
