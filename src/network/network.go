package network

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// NetworkConfig represents the configuration for a container network
type NetworkConfig struct {
	// Container ID
	ContainerID string
	// Network namespace path
	NetNSPath string
	// Bridge name
	BridgeName string
	// Subnet CIDR
	Subnet string
	// Container IP
	IP string
	// Network mode (bridge, host, none)
	Mode string
}

func init() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())
}

// SetupNetwork configures the network for a container
func SetupNetwork(config NetworkConfig) error {
	// Import utils package if not already imported
	isWSL := false
	_, wslErr := os.ReadFile("/proc/version")
	if wslErr == nil {
		content, _ := os.ReadFile("/proc/version")
		isWSL = strings.Contains(strings.ToLower(string(content)), "microsoft")
	}

	// Check for WSL environment
	if isWSL {
		fmt.Println("INFO: Running in WSL environment - network setup may be limited")

		// In WSL, some network operations might not work correctly
		// Use host networking as a fallback if bridge mode is requested
		if config.Mode == "bridge" {
			fmt.Println("WARNING: Bridge networking has limitations in WSL")
			fmt.Println("INFO: Using simplified networking implementation for WSL")
			return setupWSLNetwork(config)
		}
	}

	switch config.Mode {
	case "bridge":
		return setupBridgeNetwork(config)
	case "host":
		return setupHostNetwork(config)
	case "none":
		return nil // No networking to set up
	default:
		return fmt.Errorf("unsupported network mode: %s", config.Mode)
	}
}

// setupBridgeNetwork sets up a bridged network for the container
func setupBridgeNetwork(config NetworkConfig) error {
	// Create the bridge if it doesn't exist
	if err := ensureBridge(config.BridgeName, config.Subnet); err != nil {
		return fmt.Errorf("failed to ensure bridge: %v", err)
	}

	// Generate container IP if not specified
	containerIP := config.IP
	if containerIP == "" {
		var err error
		containerIP, err = generateIP(config.Subnet)
		if err != nil {
			return fmt.Errorf("failed to generate IP: %v", err)
		}
	}

	// Create a veth pair
	vethName := fmt.Sprintf("veth%s", truncateID(config.ContainerID))
	vethPeerName := "eth0"

	// Create the veth pair
	if err := createVethPair(vethName, vethPeerName); err != nil {
		return fmt.Errorf("failed to create veth pair: %v", err)
	}

	// Connect veth to bridge
	if err := connectVethToBridge(vethName, config.BridgeName); err != nil {
		return fmt.Errorf("failed to connect veth to bridge: %v", err)
	}

	// Move peer to container namespace
	if err := moveVethToNamespace(vethPeerName, config.NetNSPath); err != nil {
		return fmt.Errorf("failed to move veth to namespace: %v", err)
	}

	// Configure container interface
	if err := configureContainerInterface(config.NetNSPath, vethPeerName, containerIP, config.Subnet); err != nil {
		return fmt.Errorf("failed to configure container interface: %v", err)
	}

	return nil
}

// setupHostNetwork sets up host networking for the container
func setupHostNetwork(config NetworkConfig) error {
	// With host networking, the container shares the host's network namespace
	// No specific setup needed as the container will use the host's network stack
	return nil
}

// ensureBridge ensures the bridge exists and is properly configured
func ensureBridge(bridgeName, subnet string) error {
	// Check if bridge exists
	_, err := net.InterfaceByName(bridgeName)
	if err == nil {
		// Bridge exists
		return nil
	}

	// Create bridge
	cmd := exec.Command("ip", "link", "add", "name", bridgeName, "type", "bridge")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create bridge: %v", err)
	}

	// Parse subnet CIDR
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("invalid subnet CIDR: %v", err)
	}

	// Assign IP to bridge
	bridgeIP := getFirstIP(ipNet)
	cmd = exec.Command("ip", "addr", "add",
		fmt.Sprintf("%s/%d", bridgeIP, getMaskBits(ipNet.Mask)),
		"dev", bridgeName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to assign IP to bridge: %v", err)
	}

	// Bring bridge up
	cmd = exec.Command("ip", "link", "set", bridgeName, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring bridge up: %v", err)
	}

	// Enable IP forwarding
	if err := os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %v", err)
	}

	// Setup NAT
	cmd = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
		"-s", subnet, "!", "-o", bridgeName, "-j", "MASQUERADE")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to setup NAT: %v", err)
	}

	return nil
}

// createVethPair creates a veth pair
func createVethPair(vethName, peerName string) error {
	cmd := exec.Command("ip", "link", "add", vethName, "type", "veth", "peer", "name", peerName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create veth pair: %v", err)
	}
	return nil
}

// connectVethToBridge connects a veth interface to a bridge
func connectVethToBridge(vethName, bridgeName string) error {
	// Connect to bridge
	cmd := exec.Command("ip", "link", "set", vethName, "master", bridgeName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to connect to bridge: %v", err)
	}

	// Bring veth up
	cmd = exec.Command("ip", "link", "set", vethName, "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring veth up: %v", err)
	}

	return nil
}

// moveVethToNamespace moves a veth interface to a network namespace
func moveVethToNamespace(vethName, nsPath string) error {
	// Get PID from namespace path
	parts := strings.Split(nsPath, "/")
	pid := parts[len(parts)-1]

	// Move veth to namespace
	cmd := exec.Command("ip", "link", "set", vethName, "netns", pid)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to move veth to namespace: %v", err)
	}

	return nil
}

// configureContainerInterface configures the network interface inside the container
func configureContainerInterface(nsPath, ifName, ip, subnet string) error {
	// Run commands inside the namespace
	nsEnter := []string{"ip", "netns", "exec"}
	parts := strings.Split(nsPath, "/")
	pid := parts[len(parts)-1]
	nsEnter = append(nsEnter, pid)

	// Bring interface up
	cmd := exec.Command(nsEnter[0], append(nsEnter[1:], "ip", "link", "set", "dev", ifName, "up")...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bring interface up: %v", err)
	}

	// Assign IP to interface
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return fmt.Errorf("invalid subnet CIDR: %v", err)
	}

	cmd = exec.Command(nsEnter[0], append(nsEnter[1:], "ip", "addr", "add",
		fmt.Sprintf("%s/%d", ip, getMaskBits(ipNet.Mask)), "dev", ifName)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to assign IP: %v", err)
	}

	// Add default route
	gatewayIP := getFirstIP(ipNet)
	cmd = exec.Command(nsEnter[0], append(nsEnter[1:], "ip", "route", "add", "default", "via", gatewayIP)...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add default route: %v", err)
	}

	return nil
}

// generateIP generates a random IP address in the subnet
func generateIP(subnet string) (string, error) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", fmt.Errorf("invalid subnet CIDR: %v", err)
	}

	// Get network and broadcast addresses
	network := ipNet.IP
	ones, bits := ipNet.Mask.Size()
	networkSize := 1 << (bits - ones)

	// Generate a random IP in the subnet (avoiding network, gateway, and broadcast addresses)
	for {
		ip := make(net.IP, len(network))
		copy(ip, network)

		// Skip the first IP (network) and the second IP (gateway) and the last IP (broadcast)
		randNum := rand.Intn(networkSize-3) + 2

		// Apply the random number to the IP
		for i := len(ip) - 1; i >= 0; i-- {
			ip[i] = network[i] | byte(randNum&0xff)
			randNum >>= 8
		}

		if ipNet.Contains(ip) {
			return ip.String(), nil
		}
	}
}

// getFirstIP returns the first usable IP in a subnet (typically used for the gateway/bridge)
func getFirstIP(ipNet *net.IPNet) string {
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)
	ip[len(ip)-1]++
	return ip.String()
}

// truncateID truncates a container ID to the first 8 characters
func truncateID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// getMaskBits returns the number of bits in a subnet mask
func getMaskBits(mask net.IPMask) int {
	ones, _ := mask.Size()
	return ones
}

// CleanupNetwork cleans up the network configuration for a container
func CleanupNetwork(config NetworkConfig) error {
	if config.Mode != "bridge" {
		return nil
	}

	// Get the veth name
	vethName := fmt.Sprintf("veth%s", truncateID(config.ContainerID))

	// Delete the veth interface
	cmd := exec.Command("ip", "link", "del", vethName)
	// Ignore errors if the interface doesn't exist
	cmd.Run()

	return nil
}

// CreateNetworkNamespace creates a new network namespace
func CreateNetworkNamespace(containerID string) (string, error) {
	// Create namespace directory if it doesn't exist
	nsDir := "/var/run/netns"
	if err := os.MkdirAll(nsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create namespace directory: %v", err)
	}

	// Create namespace file
	nsPath := fmt.Sprintf("%s/%s", nsDir, containerID)
	cmd := exec.Command("ip", "netns", "add", containerID)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to create network namespace: %v", err)
	}

	return nsPath, nil
}

// DeleteNetworkNamespace deletes a network namespace
func DeleteNetworkNamespace(containerID string) error {
	cmd := exec.Command("ip", "netns", "del", containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete network namespace: %v", err)
	}
	return nil
}

// SetupLoopback sets up the loopback interface in a network namespace
func SetupLoopback(nsPath string) error {
	// Get PID from namespace path
	parts := strings.Split(nsPath, "/")
	pid := parts[len(parts)-1]

	// Set up loopback
	cmd := exec.Command("ip", "netns", "exec", pid, "ip", "link", "set", "lo", "up")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set up loopback: %v", err)
	}

	return nil
}
