# Makefile for the TaskLedger Go project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMODTIDY=$(GOCMD) mod tidy

# Project variables
CMD_DIR=./cmd
BINARY_NAME=taskledger
BINARY_DIR=bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)


# Default target executed when you just run `make`
all: build

# Build the application
# Creates the bin directory if it doesn't exist
build:
	@echo "Building $(BINARY_PATH) from $(CMD_DIR)..."
	@mkdir -p $(BINARY_DIR)
	$(GOBUILD) -o $(BINARY_PATH) $(CMD_DIR)
	@echo "$(BINARY_NAME) built successfully in $(BINARY_DIR)/ directory."

# Run the tests
# The -v flag provides verbose output
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Tidy up the go.mod file
tidy:
	@echo "Tidying dependencies..."
	$(GOMODTIDY)

# Clean up the built binary and the bin directory
clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -rf $(BINARY_DIR)
	@echo "Cleanup complete."

# Run the application (example with 'report' command)
# Use `make run ARGS="hours --start-date=...` for other commands
run: build
	@echo "Running $(BINARY_PATH)..."
	./$(BINARY_PATH) $(ARGS)

# Phony targets are not actual files
.PHONY: all build test clean run tidy
