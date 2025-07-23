# Makefile for container_runtime project

GO := go
BINARY_NAME := container-runtime
BUILD_DIR := build
SRC_DIR := src
SCRIPTS_DIR := scripts

.PHONY: all build clean run demo playground security socket help

all: build

build:
	@echo "Building container runtime..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)/main.go

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

run: build
	@echo "Running container runtime..."
	@sudo ./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

prepare-bundle:
	@echo "Preparing container bundle..."
	@mkdir -p bundle/rootfs/{bin,lib,proc,sys,dev,etc}
	@cp /bin/sh bundle/rootfs/bin/
	@cp examples/config.json bundle/

create: build prepare-bundle
	@echo "Creating container..."
	@sudo ./$(BUILD_DIR)/$(BINARY_NAME) -bundle bundle -id mycontainer create

start: build
	@echo "Starting container..."
	@sudo ./$(BUILD_DIR)/$(BINARY_NAME) -id mycontainer start

state: build
	@echo "Getting container state..."
	@sudo ./$(BUILD_DIR)/$(BINARY_NAME) -id mycontainer state

kill: build
	@echo "Killing container..."
	@sudo ./$(BUILD_DIR)/$(BINARY_NAME) -id mycontainer kill

delete: build
	@echo "Deleting container..."
	@sudo ./$(BUILD_DIR)/$(BINARY_NAME) -id mycontainer delete

playground:
	@echo "Running container playground demonstration..."
	@sudo ./container_playground.sh

security:
	@echo "Running container security demonstration..."
	@sudo ./container_security.sh

socket:
	@echo "Running socket/file descriptor demonstration..."
	@sudo ./socket_practice.sh

# Copy script files to scripts directory for organization
setup-scripts:
	@mkdir -p $(SCRIPTS_DIR)
	@cp container_playground.sh $(SCRIPTS_DIR)/
	@cp container_security.sh $(SCRIPTS_DIR)/
	@cp socket_practice.sh $(SCRIPTS_DIR)/
	@chmod +x $(SCRIPTS_DIR)/*.sh

help:
	@echo "Container Runtime Project Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Build the container runtime binary"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make run ARGS=\"...\"  - Run the container runtime with arguments"
	@echo "  make prepare-bundle - Prepare a simple container bundle"
	@echo "  make create        - Create a container using the prepared bundle"
	@echo "  make start         - Start the container"
	@echo "  make state         - Get container state"
	@echo "  make kill          - Kill the container"
	@echo "  make delete        - Delete the container"
	@echo "  make playground    - Run container playground demonstration"
	@echo "  make security      - Run container security demonstration"
	@echo "  make socket        - Run socket/file descriptor demonstration"
	@echo "  make setup-scripts - Copy demo scripts to scripts directory"
	@echo "  make help          - Show this help message" 