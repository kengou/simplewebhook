# Makefile for snapshot-cli
.PHONY: build clean run docker

.PHONY: build
build: build-simplewebhook

build-%:
	go build -o bin/$* ./main.go

clean:
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

docker:
	docker build -t $(BINARY_NAME):latest .

test: 
	go test ./... -coverprofile cover.out -v

## Location to install dependencies an GO binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLINT ?= $(LOCALBIN)/golangci-lint
GOLINT_VERSION ?= 2.7.2
GINKGOLINTER_VERSION ?= 0.21.2

.PHONY: lint
lint: golint
	$(GOLINT) run -v --timeout 5m

.PHONY: golint
golint: $(GOLINT)
$(GOLINT): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLINT_VERSION)
	GOBIN=$(LOCALBIN) go install github.com/nunnatsa/ginkgolinter/cmd/ginkgolinter@v$(GINKGOLINTER_VERSION)
