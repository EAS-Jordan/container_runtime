# Container Networking

Container networking is a complex topic that involves multiple Linux networking constructs to achieve isolation, connectivity, and proper performance. This document explains how container networking works under the hood and how container runtimes implement it.

## Core Concepts

Container networking has several key requirements:
1. **Isolation**: Containers need their own network namespace
2. **Connectivity**: Containers need to communicate with each other and external networks
3. **Addressing**: Each container needs its own IP address
4. **Port Management**: Services inside containers need to be accessible
5. **Network Security**: Traffic needs to be properly filtered and secured

## Network Namespaces

The fundamental building block of container networking is the Linux network namespace. A network namespace is a logical copy of the network stack with its own:
- Network interfaces
- Routing tables
- Firewall rules
- Sockets
- /proc/net directory

## Virtual Ethernet (veth) Pairs

To connect a container's network namespace to the host, container runtimes use virtual Ethernet (veth) pairs. A veth pair consists of two virtual network interfaces that act as a pipe - whatever goes in one end comes out the other.

## Network Bridge

A common pattern for container networking is to create a virtual bridge on the host, which connects all container networks:

```
                    External Network
                          |
                     Host Network
                          |
                    +------------+
                    |   Bridge   |
                    +------------+
                    /     |      \
                   /      |       \
                 veth0   veth1   veth2
                  |       |       |
            +--------+ +--------+ +--------+
            |Container| |Container| |Container|
            |    1    | |    2    | |    3    |
            +--------+ +--------+ +--------+
```

## Container Network Setup Process

Here's the typical process for setting up networking for a container:

1. Create a new network namespace for the container
2. Create a veth pair (virtual ethernet interfaces)
3. Move one end of the veth pair into the container's network namespace
4. Connect the host end of the veth pair to a bridge
5. Configure the container's network interface (assign IP address)
6. Set up routing for the container
7. Configure NAT for outbound connectivity

## Network Modes in Container Runtimes

Container runtimes typically support multiple networking modes:

### 1. Bridge Mode
The most common mode where containers connect to a virtual bridge on the host.
- Each container gets its own IP address on a private subnet
- NAT is used for outbound connections
- Port mapping is required for inbound connections

### 2. Host Mode
The container shares the host's network namespace.
- No network isolation
- No need for port mapping
- Potential port conflicts with host services

### 3. None Mode
The container gets a network namespace but no interfaces are configured.
- Complete network isolation
- No external connectivity
- Useful for security-sensitive containers

### 4. Container Mode
The container shares the network namespace with another container.
- Used in Kubernetes pods
- Containers can communicate via localhost
- Useful for sidecar patterns

## IP Address Management (IPAM)

Container runtimes need to allocate and manage IP addresses for containers:
- Keep track of allocated IP addresses
- Ensure no IP address conflicts
- Assign unique IPs to new containers
- Release IPs when containers are removed

## Network Security

Container networking includes several security aspects:

### Iptables Rules

Container runtimes often use iptables for network security and NAT:
- Port forwarding rules for container ports
- NAT masquerading for outbound traffic
- Network isolation between containers
- Security filtering

### Network Policies

More advanced container orchestration platforms like Kubernetes support network policies to control traffic between pods:
- Ingress/egress rules
- Label-based selection of pods
- CIDR-based IP filtering
- Protocol and port restrictions

## DNS for Containers

Container runtimes need to provide DNS resolution for containers:
- Configure DNS nameservers in containers
- Provide service discovery mechanisms
- Support DNS-based features like search domains

## Port Mapping

To allow external access to services running in containers:
- Map host ports to container ports
- Configure iptables DNAT rules
- Handle different protocols (TCP/UDP)
- Expose specific interfaces or all interfaces

## Multi-Host Networking

For container orchestration systems like Kubernetes, networking needs to work across multiple hosts:

### Overlay Networks

Overlay networks encapsulate container traffic to enable multi-host communication:
- VXLAN encapsulation
- VTEP (VXLAN Tunnel Endpoint) for routing
- Distributed routing tables
- Underlay network independence

## Container Network Interface (CNI)

The Container Network Interface (CNI) is a specification and libraries for writing plugins to configure network interfaces in Linux containers:

- Standard interface between runtimes and network plugins
- Multiple plugin types: main, meta, IPAM
- Chain plugins together for complex setups
- Used extensively in Kubernetes

## Real-world Container Networking

In practice, container networking is implemented by specialized networking plugins/drivers:

1. **Docker Networking**:
   - bridge
   - overlay
   - macvlan
   - ipvlan
   - none
   - host
   - plugins (like Weave, Calico, Flannel)

2. **Kubernetes Networking**:
   - Flannel
   - Calico
   - Cilium
   - Weave Net
   - AWS VPC CNI
   - Azure CNI
   - Google Kubernetes Engine networking 