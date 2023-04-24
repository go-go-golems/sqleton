.PHONY: gifs

all: gifs

VERSION=v0.1.14

TAPES=$(shell ls doc/vhs/*tape)
gifs: $(TAPES)
	for i in $(TAPES); do vhs < $$i; done

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.50.1 golangci-lint run -v

ghcr-login:
	op read "$(CR_PAT)" | docker login ghcr.io -u wesen --password-stdin

lint:
	golangci-lint run -v

test:
	go test ./...

build:
	go generate ./...
	go build ./...

sqleton:
	go build -o sqleton ./cmd/sqleton 

build-docker: sqleton
#	GOOS=linux GOARCH=amd64 go build -o sqleton ./cmd/sqleton
#	docker buildx build -t go-go-golems/sqleton:amd64 . --platform=linux/amd64
	GOOS=linux GOARCH=arm64 go build -o sqleton ./cmd/sqleton
	docker buildx build -t go-go-golems/sqleton:arm64v8 . --platform=linux/arm64/v8

up:
	docker compose up

bash:
	docker compose exec sqleton bash

goreleaser:
	goreleaser release --skip-sign --snapshot --rm-dist

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
	go build -o ./dist/sqleton ./cmd/sqleton && \
		cp ./dist/sqleton $(SQLETON_BINARY)
