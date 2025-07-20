# Control Groups (cgroups)

Control Groups (cgroups) are a Linux kernel feature that limits, accounts for, and isolates the resource usage of process groups. They are a fundamental building block for container technologies, providing the resource management aspect of containers.

## Core Concept

While namespaces provide isolation of system resources, cgroups control how much of those resources a process or group of processes can use. They allow fine-grained control over allocating, prioritizing, denying, managing, and monitoring system resources.

## Cgroups Versions

There are two major versions of cgroups with different designs:

### cgroups v1
- Introduced in Linux kernel 2.6.24 (2008)
- Each resource controller (subsystem) has its own hierarchy
- More complex but allows for more flexibility in resource management

### cgroups v2
- Introduced in Linux kernel 4.5 (2016)
- Single unified hierarchy for all resource controllers
- Simpler and more coherent design
- Modern container runtimes are transitioning to cgroups v2

## Resource Controllers (Subsystems)

Cgroups manage different types of resources through controllers:

### 1. CPU
- **cpu** - Scheduler access control
- **cpuacct** - CPU accounting
- **cpuset** - CPU pinning (restricts processes to specific CPUs)

### 2. Memory
- **memory** - Memory usage limiting and accounting

### 3. I/O
- **blkio** - Block device I/O control
- **io** (in cgroups v2) - Unified I/O controller

### 4. Devices
- **devices** - Device access control

### 5. Network
- **net_cls** - Network packet tagging
- **net_prio** - Network priority

### 6. Other
- **pids** - Process number limitation
- **freezer** - Suspend/resume for groups of processes
- **hugetlb** - Huge pages management

## Cgroups Filesystem Structure

### v1 Structure
In cgroups v1, each subsystem is mounted separately:

```
/sys/fs/cgroup/
├── cpu/
├── memory/
├── blkio/
├── devices/
└── ...
```

### v2 Structure
In cgroups v2, there's a unified hierarchy:

```
/sys/fs/cgroup/
├── cgroup.controllers
├── cgroup.subtree_control
├── container1/
│   ├── cgroup.controllers
│   ├── memory.max
│   ├── memory.current
│   ├── cpu.max
│   └── ...
└── container2/
    └── ...
```

## Creating and Managing cgroups

Creating and managing cgroups involves:
1. Creating a directory under `/sys/fs/cgroup/` or the appropriate controller directory
2. Setting control parameters by writing to files in that directory
3. Adding processes to the cgroup by writing their PIDs to the `cgroup.procs` file
4. For cgroups v2, enabling controllers by writing to the `cgroup.subtree_control` file

## Container Resource Limits

Here's how cgroups are used to implement container resource limits:

### Memory Limits
- **memory.max**: Hard limit of memory usage
- **memory.high**: Soft limit, triggering reclaim but not OOM killer
- **memory.low**: Protection from aggressive reclaim
- **memory.swap.max**: Swap usage limit

### CPU Limits
- **cpu.max**: Sets CPU usage quota/period
- **cpu.weight**: Sets relative CPU shares
- **cpuset.cpus**: Restricts to specific CPU cores

### I/O Limits
- **io.max**: Limits IOPS or bandwidth
- **io.weight**: Sets relative I/O weight

## Relationship with Container Runtimes

1. **Docker**: Docker uses cgroups to enforce resource constraints defined with flags like `--memory`, `--cpu-quota`, and `--cpuset-cpus`

2. **containerd/runc**: These lower-level runtimes directly manage the cgroup configurations based on OCI container specs

3. **Kubernetes**: Uses cgroups for enforcing resource requests and limits for pods and containers

## Monitoring and Accounting

Cgroups provide valuable metrics for container monitoring through files in the cgroup filesystem:
- Memory usage: `memory.current`, `memory.peak`
- CPU usage: `cpu.stat`
- I/O statistics: `io.stat`

## Advanced Topics

### Pressure Stall Information (PSI)
- Provides information about resource pressure (CPU, memory, I/O)
- Available in cgroups v2
- Useful for dynamic resource management

### Delegation
- cgroups v2 supports delegating control to unprivileged users
- Enables rootless containers

### OOM Control
- **memory.oom.group**: When set to 1, the OOM killer treats all processes in the cgroup as a single unit
- **oom_score_adj**: Influences which processes get killed first under memory pressure

## Comparison to Other Resource Management Systems

- **Linux Containers (LXC)**: Uses cgroups extensively for resource control
- **systemd**: Uses cgroups to manage services and their resource usage
- **Solaris Zones/Containers**: Similar concept but predates cgroups
- **FreeBSD Jails**: Resource limits implemented differently

## Future Developments

- Complete transition from cgroups v1 to v2
- More fine-grained control over resources
- Better integration with eBPF for advanced observability
- Further improvements to delegation and unprivileged usage 