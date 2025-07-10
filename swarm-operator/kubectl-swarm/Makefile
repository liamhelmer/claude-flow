# Copyright 2024 The Swarm Authors.
# Licensed under the Apache License, Version 2.0

BINARY_NAME=kubectl-swarm
VERSION?=0.1.0
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
BUILDDIR=dist

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildDate=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')"

.PHONY: all build clean test deps build-all install krew-manifest

all: test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -rf $(BUILDDIR)
	rm -f $(BINARY_NAME)

deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Cross compilation
build-all: clean
	@mkdir -p $(BUILDDIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$$(echo $$platform | cut -d/ -f1) \
		GOARCH=$$(echo $$platform | cut -d/ -f2) \
		output=$(BUILDDIR)/$(BINARY_NAME)-$$(echo $$platform | sed 's/\//-/g'); \
		if [ $$GOOS = "windows" ]; then output="$$output.exe"; fi; \
		echo "Building $$output..."; \
		GOOS=$$GOOS GOARCH=$$GOARCH $(GOBUILD) $(LDFLAGS) -o $$output; \
	done

# Local installation
install: build
	@echo "Installing kubectl-swarm..."
	@mkdir -p ~/.local/bin
	@cp $(BINARY_NAME) ~/.local/bin/
	@echo "Installation complete. Make sure ~/.local/bin is in your PATH"

# Generate Krew manifest
krew-manifest:
	@echo "Generating Krew manifest..."
	@mkdir -p $(BUILDDIR)/krew
	@cat hack/krew-template.yaml | \
		sed 's/{{VERSION}}/$(VERSION)/g' > $(BUILDDIR)/krew/swarm.yaml

# Create release artifacts
release: build-all krew-manifest
	@echo "Creating release artifacts..."
	@cd $(BUILDDIR) && \
	for file in kubectl-swarm-*; do \
		tar -czf $$file.tar.gz $$file; \
		sha256sum $$file.tar.gz > $$file.tar.gz.sha256; \
	done

# Run go fmt
fmt:
	$(GOCMD) fmt ./...

# Run go vet
vet:
	$(GOCMD) vet ./...

# Run golangci-lint
lint:
	golangci-lint run

# Generate code
generate:
	$(GOCMD) generate ./...