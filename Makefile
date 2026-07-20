.PHONY: build test tidy help

BINARY := bin/pr-agent
GO_IMAGE := golang:1.25

help:
	@echo "Targets: build, test, tidy"

build:
	docker run --rm -v "$(CURDIR)":/app -w /app $(GO_IMAGE) go build -o $(BINARY) ./cmd/pr-agent

test:
	docker run --rm -v "$(CURDIR)":/app -w /app $(GO_IMAGE) go test ./...

tidy:
	docker run --rm -v "$(CURDIR)":/app -w /app $(GO_IMAGE) go mod tidy
