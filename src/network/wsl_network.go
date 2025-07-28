package network

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/container-runtime/core/utils"
)

// setupWSLNetwork provides a simplified network setup for WSL environments
// It uses a combination of host networking and minimal namespace setup
func setupWSLNetwork(config NetworkConfig) error {
	// Check if we're in WSL
	if !utils.IsWSL() {
		// Not in WSL, use standard bridge networking
		return setupBridgeNetwork(config)
	}

	fmt.Println("INFO: Setting up WSL-compatible networking")

	// In WSL, we'll set up a simplified network configuration:
	// 1. We'll skip creating a network bridge (it might not work well)
	// 2. We'll create a network namespace just for isolation
	// 3. We'll set up a loopback device inside the namespace
	// 4. We'll provide access to the host network via WSL's eth0

	// First check if the network namespace path exists
	if config.NetNSPath == "" {
		// Create a namespace if not provided
		var err error
		config.NetNSPath, err = CreateNetworkNamespace(config.ContainerID)
		if err != nil {
			fmt.Printf("WARNING: Failed to create network namespace: %v\n", err)
			fmt.Println("INFO: Continuing with host networking")
			return nil // Fallback to host networking
		}
	}

	// Set up loopback interface
	if err := SetupLoopback(config.NetNSPath); err != nil {
		fmt.Printf("WARNING: Failed to set up loopback: %v\n", err)
		// Continue anyway, this isn't critical
	}

	// Check if we can access WSL's primary interface (usually eth0)
	wslInterface := getWSLPrimaryInterface()
	if wslInterface == "" {
		fmt.Println("WARNING: Could not determine WSL primary interface")
		fmt.Println("INFO: Container will have limited network connectivity")
		return nil
	}

	// Get PID from namespace path for use with ip netns commands
	parts := strings.Split(config.NetNSPath, "/")
	pid := parts[len(parts)-1]

	// Try to share WSL's eth0 connection with the container
	// This is a simplified approach that might work in some WSL configurations
	fmt.Printf("INFO: Attempting to configure container access to WSL network via %s\n", wslInterface)

	// Rather than creating a veth pair, which might not work well in WSL,
	// we'll create a simple NAT setup for the container namespace
	setupNATCommands := [][]string{
		// Enable IP forwarding (might already be enabled in WSL)
		{"sysctl", "-w", "net.ipv4.ip_forward=1"},

		// Add a route from the container namespace to the host
		{"ip", "netns", "exec", pid, "ip", "route", "add", "default", "via", "172.17.0.1"},

		// Set up NAT for outbound traffic
		{"iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "172.17.0.0/16", "-o", wslInterface, "-j", "MASQUERADE"},
	}

	for _, cmd := range setupNATCommands {
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		output, err := execCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("WARNING: Command '%s' failed: %v\n%s\n",
				strings.Join(cmd, " "), err, string(output))
			// Continue anyway, this is best-effort in WSL
		}
	}

	fmt.Println("INFO: WSL network setup completed with best-effort configuration")
	fmt.Println("NOTE: Some network features may be limited in WSL")
	return nil
}

// getWSLPrimaryInterface tries to detect WSL's primary network interface
func getWSLPrimaryInterface() string {
	// Common WSL interfaces
	possibleInterfaces := []string{"eth0", "wsl", "wslbr"}

	// Try to find an existing interface
	for _, iface := range possibleInterfaces {
		if interfaceExists(iface) {
			return iface
		}
	}

	// Fallback: look for any interface with a default route
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.CombinedOutput()
	if err == nil {
		outputStr := string(output)
		// Parse the default route output to extract interface
		for _, line := range strings.Split(outputStr, "\n") {
			if strings.HasPrefix(line, "default") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == "dev" && i+1 < len(fields) {
						return fields[i+1]
					}
				}
			}
		}
	}

	return ""
}

// interfaceExists checks if a network interface exists
func interfaceExists(name string) bool {
	_, err := os.Stat(fmt.Sprintf("/sys/class/net/%s", name))
	return err == nil
}
