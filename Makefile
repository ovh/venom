include linting.mk

.PHONY: all
all: deps lint build test

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: download
download:
	go mod download

.PHONY: deps
deps: tidy download

.PHONY: verify
verify:
	go mod verify

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./...
