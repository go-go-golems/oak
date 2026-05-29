.PHONY: lint lintmax gosec govulncheck test build goreleaser tag-major tag-minor tag-patch release bump-glazed install

all: 

VERSION=v0.1.0

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.3.0 golangci-lint run -v

lint:
	golangci-lint run -v

lintmax:
	golangci-lint run -v --max-same-issues=100

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G101,G304,G301,G306,G204 -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	go test ./...

build:
	go generate ./...
	go build ./...

goreleaser:
	goreleaser release --skip=sign --snapshot --clean

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/oak@$(shell svu current)

bump-glazed:
	go get github.com/go-go-golems/glazed@latest
	go get github.com/go-go-golems/clay@latest
	go get github.com/go-go-golems/bobatea@latest
	go mod tidy

OAK_BINARY=$(shell which oak)
install:
	go build -o ./dist/oak ./cmd/oak && \
		cp ./dist/oak $(OAK_BINARY)

.PHONY: logcopter-generate
logcopter-generate:
	GOWORK=off go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.oak -strip-prefix github.com/go-go-golems/oak ./cmd/... ./pkg/...

.PHONY: logcopter-check
logcopter-check:
	GOWORK=off go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.oak -strip-prefix github.com/go-go-golems/oak -check ./cmd/... ./pkg/...

GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= v0.6.12

.PHONY: glazed-lint-build glazed-lint

glazed-lint-build:
	@echo "Building glazed-lint from Glazed module..."
	@if [ -n "$(GLAZED_VERSION)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION); \
	else \
		echo "Installing $(GLAZED_LINT_PKG) from workspace/module"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) go install $(GLAZED_LINT_PKG); \
	fi

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) ./cmd/... ./pkg/...
