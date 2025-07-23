package utils

import (
	"bufio"
	"os"
	"strings"
)

// IsWSL returns true if running in Windows Subsystem for Linux
func IsWSL() bool {
	// Check for /proc/version containing Microsoft
	file, err := os.Open("/proc/version")
	if err != nil {
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), "microsoft") {
			return true
		}
	}

	// As a secondary check, look for WSL-specific environment variables
	_, hasWSLEnv := os.LookupEnv("WSL_DISTRO_NAME")
	_, hasWSLInterop := os.LookupEnv("WSL_INTEROP")

	return hasWSLEnv || hasWSLInterop
}

// FeatureSupport represents the level of support for a container feature in WSL
type FeatureSupport int

const (
	Supported FeatureSupport = iota
	LimitedSupport
	Unsupported
)

// CheckFeatureSupport checks if a particular container feature is supported in the current environment
func CheckFeatureSupport(feature string) FeatureSupport {
	isWsl := IsWSL()

	if !isWsl {
		return Supported // Assume full support if not in WSL
	}

	// Define support levels for various features in WSL
	wslSupport := map[string]FeatureSupport{
		"pid_namespace":      Supported,      // PID namespace works in WSL2
		"uts_namespace":      Supported,      // UTS namespace works in WSL2
		"mount_namespace":    LimitedSupport, // Mount namespace has limitations in WSL2
		"net_namespace":      LimitedSupport, // Network namespace has limitations
		"user_namespace":     LimitedSupport, // User namespace may have restrictions
		"ipc_namespace":      Supported,      // IPC namespace works in WSL2
		"cgroups_v1":         LimitedSupport, // Limited support for cgroups v1
		"cgroups_v2":         LimitedSupport, // Limited support for cgroups v2
		"overlayfs":          Supported,      // OverlayFS should work in WSL2
		"bind_mounts":        Supported,      // Bind mounts work in WSL2
		"procfs_mounts":      Supported,      // procfs mounts work in WSL2
		"sysfs_mounts":       Supported,      // sysfs mounts work in WSL2
		"devpts_mounts":      Supported,      // devpts mounts work in WSL2
		"device_creation":    LimitedSupport, // Device creation may be limited
		"seccomp":            Unsupported,    // Seccomp is not well supported in WSL2
		"capabilities":       LimitedSupport, // Capabilities may have limitations
		"pivot_root":         LimitedSupport, // pivot_root might have issues in WSL2
		"veth":               LimitedSupport, // Virtual ethernet has limitations
		"bridge_networking":  LimitedSupport, // Bridge networking has limitations
		"iptables":           LimitedSupport, // iptables has limitations in WSL2
		"privileged_actions": LimitedSupport, // Privileged operations may be restricted
	}

	support, exists := wslSupport[feature]
	if !exists {
		return LimitedSupport // Default to limited support for unknown features
	}

	return support
}

// GetWSLWarningMessage returns a warning message for a specific feature in WSL
func GetWSLWarningMessage(feature string) string {
	messages := map[string]string{
		"mount_namespace":    "Mount namespace operations may be limited in WSL2",
		"net_namespace":      "Network namespace operations may have limited functionality in WSL2",
		"user_namespace":     "User namespace operations might be restricted in WSL2",
		"cgroups_v1":         "Cgroups v1 has limited support in WSL2",
		"cgroups_v2":         "Cgroups v2 has limited support in WSL2",
		"device_creation":    "Device node creation may be restricted in WSL2",
		"seccomp":            "Seccomp filters are not well supported in WSL2",
		"capabilities":       "Capability operations may be restricted in WSL2",
		"pivot_root":         "pivot_root may not work correctly in WSL2, consider using chroot instead",
		"veth":               "Virtual ethernet setup might require additional configuration in WSL2",
		"bridge_networking":  "Bridge networking might not work as expected in WSL2",
		"iptables":           "iptables rules might not work correctly in WSL2",
		"privileged_actions": "Some privileged operations may be restricted in WSL2",
	}

	message, exists := messages[feature]
	if !exists {
		return "This feature may have limited support in WSL2"
	}

	return message
}

// TryWSLCompat attempts to make a feature work in WSL by applying workarounds
func TryWSLCompat(feature string) (worked bool, message string) {
	if !IsWSL() {
		return true, "" // Not in WSL, no compatibility needed
	}

	switch feature {
	case "pivot_root":
		// In WSL, pivot_root might not work, suggest using chroot instead
		return false, "pivot_root is not well supported in WSL2. Consider using chroot instead."

	case "mount_namespace":
		// Some mounts might need special handling in WSL
		return true, "Mount namespace is available but some mount operations may fail in WSL2."

	case "net_namespace":
		// Network namespace exists but might need special handling
		return true, "Network namespace is available but network setup might need manual configuration in WSL2."

	case "cgroups_v2":
		// Check if cgroups v2 is available in this WSL instance
		if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
			return true, "Cgroups v2 appears to be available but may have limited functionality."
		}
		return false, "Cgroups v2 unified hierarchy not available in this WSL2 instance."

	default:
		// For other features, just return the default warning
		support := CheckFeatureSupport(feature)
		if support == Supported {
			return true, ""
		}
		return support != Unsupported, GetWSLWarningMessage(feature)
	}
}
