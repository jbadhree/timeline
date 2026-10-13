# Local and CI checks. `make check` is the one command to run before pushing.

GOLANGCI_LINT_VERSION := v2.14.0
BIN := $(CURDIR)/bin
GOLANGCI_LINT := $(BIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

.PHONY: check fmt-check build lint test race plan

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

# Validate this repository's own goals and tasks. Not part of `check`: drafts
# are checked against the current time, so the result changes as dates pass.
plan:
	go run ./cmd/tline check --dry-run

$(GOLANGCI_LINT):
	mkdir -p $(BIN)
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	mv $(BIN)/golangci-lint $@
