# Container Runtime Implementation Project

This project contains educational materials and implementation code for understanding container technologies from the ground up. It demonstrates how Linux kernel features are leveraged to create container runtimes similar to Docker and containerd.

## Educational Purpose

These materials are designed to teach engineers about:
- How containers work under the hood
- Core Linux kernel features that enable containerization
- Container runtime implementation details
- Kubernetes pod architecture

## Core Concepts Covered

1. **Linux Kernel Namespaces** - Isolation mechanisms for processes
2. **Control Groups (cgroups)** - Resource limiting and accounting
3. **UnionFS/OverlayFS** - Filesystem layering for container images
4. **Networking** - Virtual interfaces and network namespaces
5. **Linux File Descriptors** - The "everything is a file" philosophy
6. **OCI Container Specification** - Standards-based container implementation

## Project Structure

```
container_runtime/
├── docs/                           # Educational materials
│   ├── linux_namespaces.md         # Linux namespaces explained
│   ├── cgroups.md                  # Control groups (cgroups) explained
│   ├── overlay_filesystem.md       # UnionFS/OverlayFS explained
│   ├── container_networking.md     # Container networking explained
│   ├── linux_file_descriptors.md   # Linux file descriptor system explained
│   └── kubernetes_pods.md          # Kubernetes pod architecture explained
├── src/                            # Implementation code
│   ├── main.go                     # Entry point with CLI commands
│   ├── runtime/                    # OCI runtime implementation
│   │   └── container.go            # Container lifecycle management
│   ├── namespace/                  # Namespace operations
│   │   └── namespace.go            # Linux namespace handling 
│   ├── cgroups/                    # cgroup operations
│   │   └── cgroups.go              # Resource limit management
│   ├── fs/                         # Filesystem operations
│   │   └── overlay.go              # OverlayFS implementation
│   ├── network/                    # Network setup operations
│   │   └── network.go              # Network namespace and interface setup
│   └── pod/                        # Kubernetes pod implementation
│       └── pod.go                  # Pod lifecycle management
└── examples/                       # Example configurations
    ├── config.json                 # Sample OCI container config
    └── pod.json                    # Sample pod definition
```

## Getting Started

To learn about containers:
1. Start with the educational materials in the `docs/` directory
2. Explore the implementation code in the `src/` directory
3. Run the interactive demos to understand container fundamentals
4. Build and run the container runtime to see the concepts in action

### Interactive Demonstrations

This project includes several interactive demonstrations to help understand container fundamentals:

```bash
# Run the container playground demo (basic namespaces and isolation)
make playground

# Run the container security demo (security concepts and risks)
make security

# Run the socket practice demo (file descriptors and Unix sockets)
make socket
```

```bash
# Build the container runtime
cd container_runtime
go build -o container-runtime src/main.go

# Create an OCI bundle
mkdir -p mybundle/rootfs
cp examples/config.json mybundle/

# Create a rootfs (in a real scenario, you would use a real container image)
# This is just a simplified example
mkdir -p mybundle/rootfs/{bin,lib,proc,sys,dev,etc}
cp /bin/sh mybundle/rootfs/bin/

# Create a container
sudo ./container-runtime -bundle mybundle -id mycontainer create

# Start the container
sudo ./container-runtime -id mycontainer start

# Get container state
sudo ./container-runtime -id mycontainer state

# Stop and delete the container
sudo ./container-runtime -id mycontainer kill
sudo ./container-runtime -id mycontainer delete
```

## Pod Support

For Kubernetes-like pod functionality:

```bash
# Create a pod from a pod definition
sudo ./container-runtime pod create -f examples/pod.json

# List pods
sudo ./container-runtime pod list

# Start a pod
sudo ./container-runtime pod start mypod

# Stop a pod
sudo ./container-runtime pod stop mypod

# Delete a pod
sudo ./container-runtime pod delete mypod
```

## WSL Compatibility

This project now includes compatibility improvements for running in Windows Subsystem for Linux (WSL2):

### Limitations in WSL

While WSL2 provides a Linux kernel that supports most container features, some aspects may be limited:

1. **Namespace Support**: Most namespaces work in WSL2, but with some edge cases
2. **cgroups**: Limited support for cgroup operations
3. **Networking**: Bridge networking may require special configuration
4. **pivot_root**: May not work correctly in all WSL environments
5. **Privileged Operations**: Some privileged operations may be restricted

### WSL Compatibility Features

The codebase includes several adaptations for better WSL compatibility:

1. **Automatic Detection**: The code detects when running in WSL
2. **Graceful Fallbacks**: When a feature doesn't work, the code uses alternatives:
   - Using `chroot` instead of `pivot_root`
   - Simulating cgroups when not available
   - Simplified networking setup
3. **Helpful Warnings**: The code provides informative messages about WSL limitations

### Running in WSL

When running in WSL, you'll see WSL-specific informational messages and warnings. The code will attempt to use compatible alternatives when standard features are unavailable.

For the best experience with full container features, consider using a standard Linux environment or VM.

## Requirements

- Linux system with kernel 4.0+ (5.0+ recommended)
- Go 1.16+
- Root privileges (for namespace and cgroup operations)

## License

This project is provided for educational purposes. 
