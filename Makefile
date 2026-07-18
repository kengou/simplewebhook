# Makefile for simplewebhook

BINARY_NAME := simplewebhook

.PHONY: build clean run docker test lint golint

build:
	go build -o bin/$(BINARY_NAME) ./main.go

clean:
	rm -f bin/$(BINARY_NAME)

run: build
	./bin/$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):latest .

test:
	go test ./... -coverprofile cover.out -v

## Location to install dependencies an GO binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLINT ?= $(LOCALBIN)/golangci-lint
GOLINT_VERSION ?= 2.12.2

lint: golint
	$(GOLINT) run -v --timeout 5m

golint: $(GOLINT)
$(GOLINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLINT_VERSION)
