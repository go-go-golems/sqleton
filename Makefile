.PHONY: gifs test build lint lintmax docker-lint gosec govulncheck goreleaser tag-major tag-minor tag-patch release bump-glazed install codeql-local

VERSION ?= $(shell svu)
COMMIT ?= $(shell git rev-parse --short HEAD)
DIRTY ?= $(shell git diff --quiet || echo "dirty")
LDFLAGS=-ldflags "-X main.version=$(VERSION)-$(COMMIT)-$(DIRTY)"

all: test build

TAPES=$(shell ls doc/vhs/*tape)
gifs: $(TAPES)
	for i in $(TAPES); do vhs < $$i; done

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v2.3.0 golangci-lint run -v

ghcr-login:
	op read "$(CR_PAT)" | docker login ghcr.io -u wesen --password-stdin

lint:
	golangci-lint run -v

lintmax:
	golangci-lint run -v --max-same-issues=100

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	# Adjust exclusions as needed
	gosec -exclude=G101,G304,G301,G306,G204 -exclude-dir=.history ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	go test ./...

build:
	go generate ./...
	go build $(LDFLAGS) ./...

sqleton:
	go build $(LDFLAGS) -o sqleton ./cmd/sqleton

build-docker: sqleton
#	GOOS=linux GOARCH=amd64 go build -o sqleton ./cmd/sqleton
#	docker buildx build -t go-go-golems/sqleton:amd64 . --platform=linux/amd64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o sqleton ./cmd/sqleton
	docker buildx build -t go-go-golems/sqleton:arm64v8 . --platform=linux/arm64/v8

up:
	docker compose up

bash:
	docker compose exec sqleton bash

goreleaser:
	goreleaser release --skip=sign --snapshot --clean

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/sqleton@$(shell svu current)

bump-glazed:
	go get github.com/go-go-golems/glazed@latest
	go get github.com/go-go-golems/clay@latest
	go get github.com/go-go-golems/parka@latest
	go mod tidy

SQLETON_BINARY=$(shell which sqleton)
install:
	go build $(LDFLAGS) -o ./dist/sqleton ./cmd/sqleton && \
		cp ./dist/sqleton $(SQLETON_BINARY)

# Path to CodeQL CLI - adjust based on installation location
CODEQL_PATH ?= $(shell which codeql)
# Path to CodeQL queries - adjust based on where you cloned the repository
CODEQL_QUERIES ?= $(HOME)/codeql-go/ql/src/go

# Create CodeQL database and run analysis
codeql-local:
	@if [ -z "$(CODEQL_PATH)" ]; then echo "CodeQL CLI not found. Install from https://github.com/github/codeql-cli-binaries/releases"; exit 1; fi
	@if [ ! -d "$(CODEQL_QUERIES)" ]; then echo "CodeQL queries not found. Clone from https://github.com/github/codeql-go"; exit 1; fi
	$(CODEQL_PATH) database create --language=go --source-root=. ./codeql-db
	$(CODEQL_PATH) database analyze ./codeql-db $(CODEQL_QUERIES)/Security --format=sarif-latest --output=codeql-results.sarif
	@echo "Results saved to codeql-results.sarif"
