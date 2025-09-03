# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=kori
BINARY_UNIX=$(BINARY_NAME)_unix

# Build parameters
BUILD_DIR=build
MAIN_PATH=cmd/main.go

.PHONY: all build test clean run deps dev docs docs-serve docs-clean openapi

all: test build

build:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PATH)

test:
	$(GOTEST) -v ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run:
	nodemon -w . -e go,yaml,toml --exec "make build"

deps:
	$(GOMOD) download
	$(GOMOD) verify

# Swagger/OpenAPI documentation commands
docs:
	@echo "Generating OpenAPI/Swagger documentation..."
	@export PATH=$$PATH:$$(go env GOPATH)/bin && swag init -g cmd/main.go -o docs/swagger --parseDependency --parseInternal
	@echo "✅ Documentation generated successfully!"

docs-serve:
	@echo "Starting Swagger UI server..."
	@export PATH=$$PATH:$$(go env GOPATH)/bin && swag serve -F=swagger docs/swagger/swagger.json

docs-clean:
	@echo "Cleaning generated documentation..."
	@rm -rf docs/swagger/*
	@echo "✅ Documentation cleaned!"

# OpenAPI 3.0 specification
openapi:
	@echo "Generating OpenAPI 3.0 specification..."
	@./scripts/generate-openapi.sh generate
	@echo "✅ OpenAPI 3.0 specification generated as openapi.json!"

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(MAIN_PATH)

dev:
	nodemon

helper:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_UNIX) -v cmd/helper/main.go
