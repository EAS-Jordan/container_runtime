#!/bin/bash
# setup.sh - Setup script for the container runtime project

set -e

echo "======= CONTAINER RUNTIME PROJECT SETUP ======="
echo "This script will set up the container runtime project and run a quick demo"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go first."
    exit 1
fi

# Check if running as root (required for container operations)
if [ "$(id -u)" -ne 0 ]; then
    echo "This script must be run as root"
    exit 1
fi

echo "1. Setting up project structure..."
mkdir -p build bundle/rootfs

echo "2. Installing dependencies..."
go mod tidy

echo "3. Building the container runtime..."
make build

echo "4. Setting up a minimal container bundle..."
mkdir -p bundle/rootfs/{bin,lib,lib64,proc,sys,dev,etc,tmp}

# Copy essential binaries and dependencies
cp /bin/sh bundle/rootfs/bin/

# Copy config.json if it doesn't exist
if [ ! -f bundle/config.json ]; then
    cp examples/config.json bundle/
fi

echo "5. Creating container..."
./build/container-runtime -bundle bundle -id demo-container create

echo "6. Getting container state..."
./build/container-runtime -id demo-container state

echo "7. Starting container..."
./build/container-runtime -id demo-container start

echo "8. Checking container state again..."
./build/container-runtime -id demo-container state

echo "9. Stopping container..."
./build/container-runtime -id demo-container kill

echo "10. Cleaning up..."
./build/container-runtime -id demo-container delete

echo "======= CONTAINER RUNTIME PROJECT SETUP COMPLETE ======="
echo
echo "Now you can:"
echo "- Run the container playground demo: sudo ./container_playground.sh"
echo "- Run the security demonstration: sudo ./container_security.sh"
echo "- Run the socket demonstration: sudo ./socket_practice.sh"
echo "- Explore the code in the src/ directory"
echo "- Read the documentation in docs/ directory"
echo
echo "Use 'make help' to see all available commands" 