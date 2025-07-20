# Linux Namespaces

Linux namespaces are a fundamental kernel feature that enables process isolation, which forms the core of container technology. Namespaces partition kernel resources such that one set of processes sees one set of resources while another set sees a different set.

## Core Concept

Namespaces create isolated contexts for various system resources, making processes within a namespace appear to have their own isolated instance of the global resource.

## Types of Namespaces Used in Containers

### 1. PID Namespace (Process ID)
- **What it does**: Isolates the process ID number space
- **Effect**: Processes in different PID namespaces can have the same PID
- **Container relevance**: The container's init process appears as PID 1 inside the container
- **Linux syscall flag**: `CLONE_NEWPID`

### 2. Mount Namespace (mnt)
- **What it does**: Isolates filesystem mount points
- **Effect**: Processes in different mount namespaces have different views of the filesystem hierarchy
- **Container relevance**: Enables containers to have their own root filesystem
- **Linux syscall flag**: `CLONE_NEWNS`

### 3. Network Namespace (net)
- **What it does**: Isolates network interfaces, routing tables, firewall rules, etc.
- **Effect**: Each namespace can have independent network devices and configurations
- **Container relevance**: Allows containers to have their own IP address, ports, routing tables
- **Linux syscall flag**: `CLONE_NEWNET`

### 4. UTS Namespace (Unix Time Sharing)
- **What it does**: Isolates hostname and domain name
- **Effect**: Processes in different UTS namespaces can have different hostnames
- **Container relevance**: Allows containers to have their own hostname
- **Linux syscall flag**: `CLONE_NEWUTS`

### 5. IPC Namespace (Inter-Process Communication)
- **What it does**: Isolates IPC resources (shared memory segments, semaphores, message queues)
- **Effect**: Processes in different IPC namespaces cannot use IPC mechanisms to communicate
- **Container relevance**: Prevents IPC between processes in different containers
- **Linux syscall flag**: `CLONE_NEWIPC`

### 6. User Namespace (user)
- **What it does**: Isolates user and group ID number spaces
- **Effect**: Allows unprivileged users to run processes that have root privileges inside the namespace
- **Container relevance**: Enhances container security by limiting privileges outside the container
- **Linux syscall flag**: `CLONE_NEWUSER`

## How Namespaces Are Created

Namespaces are created through system calls like:
- `clone()` - Creates a new process and a new namespace
- `unshare()` - Disassociates parts of a process's execution context
- `setns()` - Allows a process to join an existing namespace

## Viewing Namespaces in Linux

You can view the namespaces a process belongs to:
```bash
ls -la /proc/<pid>/ns/
```

## Implications for Container Security

Namespaces provide isolation but not security by themselves:
- Root in a container with default settings can still be dangerous
- User namespaces add a layer of security by remapping UIDs
- Additional security measures like seccomp filters, capabilities, and SELinux/AppArmor are often used alongside namespaces

## Limitations of Namespace Isolation

- **Kernel is shared**: All containers on a host share the same kernel
- **Resource limits not enforced**: Namespaces only provide isolation, not resource constraints (that's what cgroups are for)
- **Some kernel resources are not namespaced**: For example, some `/proc` and `/sys` entries, device nodes, kernel modules

## Relationship to Other Container Technologies

Namespaces provide the isolation aspect of containers, while other technologies handle different aspects:
- **cgroups**: Resource limitation
- **UnionFS/OverlayFS**: Filesystem layering
- **Seccomp/capabilities**: Security constraints

## Advanced Topics

- **Nested Namespaces**: Namespaces can be nested to create complex hierarchies
- **Namespace Inheritance**: Child processes inherit their parent's namespaces by default
- **Namespace Lifetime**: A namespace exists as long as a process is using it or it has a bind mount reference 