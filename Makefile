# Root Makefile for Card Onboarding Platform

.PHONY: all generate test lint clean

all: generate build test

generate:
	@echo "Generating API models and clients..."
	cd card-onboarding-services && $(MAKE) generate

test:
	@echo "Running unit tests across all Go modules..."
	go test ./...

lint:
	@echo "Linting all packages in the workspace..."
	golangci-lint run ./...

clean:
	@echo "Cleaning compiled binaries..."
	rm -rf card-onboarding-workers/bin/
