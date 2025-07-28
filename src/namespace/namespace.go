package namespace

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/container-runtime/core/utils"
)

// Define namespace types
const (
	NEWUTS  = syscall.CLONE_NEWUTS  // UTS namespace
	NEWIPC  = syscall.CLONE_NEWIPC  // IPC namespace
	NEWPID  = syscall.CLONE_NEWPID  // PID namespace
	NEWNS   = syscall.CLONE_NEWNS   // Mount namespace
	NEWNET  = syscall.CLONE_NEWNET  // Network namespace
	NEWUSER = syscall.CLONE_NEWUSER // User namespace
)

// Setup creates all required namespaces for a container
func Setup(hostname string, usePID, useIPC, useUTS, useNet, useMount bool) error {
	// Verify we're on Linux
	if runtime.GOOS != "linux" {
		return fmt.Errorf("namespace operations require Linux")
	}

	// Check for WSL compatibility
	if utils.IsWSL() {
		log.Println("Running in WSL environment - some namespace operations may be limited")
	}

	// Create a namespace bitmask based on the flags
	var namespaceFlags uintptr
	if usePID {
		if worked, msg := utils.TryWSLCompat("pid_namespace"); !worked {
			return fmt.Errorf("PID namespace issue: %s", msg)
		}
		namespaceFlags |= NEWPID
	}
	if useIPC {
		if worked, msg := utils.TryWSLCompat("ipc_namespace"); !worked {
			return fmt.Errorf("IPC namespace issue: %s", msg)
		}
		namespaceFlags |= NEWIPC
	}
	if useUTS {
		if worked, msg := utils.TryWSLCompat("uts_namespace"); !worked {
			return fmt.Errorf("UTS namespace issue: %s", msg)
		}
		namespaceFlags |= NEWUTS
	}
	if useNet {
		if worked, msg := utils.TryWSLCompat("net_namespace"); !worked {
			log.Printf("WARNING: %s", msg)
			log.Println("Continuing without network namespace isolation")
			// We don't return error here to allow partial functionality
		} else if msg != "" {
			log.Printf("NOTE: %s", msg)
		}
		namespaceFlags |= NEWNET
	}
	if useMount {
		if worked, msg := utils.TryWSLCompat("mount_namespace"); !worked {
			log.Printf("WARNING: %s", msg)
			log.Println("Continuing without mount namespace isolation")
			// We don't return error here to allow partial functionality
		} else if msg != "" {
			log.Printf("NOTE: %s", msg)
		}
		namespaceFlags |= NEWNS
	}

	// Create new namespaces
	if err := syscall.Unshare(int(namespaceFlags)); err != nil {
		// Try individual namespace creation if the combined unshare fails
		if utils.IsWSL() {
			log.Println("Combined unshare failed, trying individual namespace creation...")
			return setupNamespacesIndividually(hostname, usePID, useIPC, useUTS, useNet, useMount)
		}
		return fmt.Errorf("failed to unshare namespaces: %v", err)
	}

	// If we're in a UTS namespace, set the hostname
	if useUTS && hostname != "" {
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			return fmt.Errorf("failed to set hostname: %v", err)
		}
	}

	// If we're in a PID namespace, mount a new proc filesystem
	if usePID {
		// Make sure /proc exists
		if err := os.MkdirAll("/proc", 0755); err != nil {
			return fmt.Errorf("failed to create /proc directory: %v", err)
		}

		// Mount proc filesystem
		if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
			if utils.IsWSL() {
				log.Println("WARNING: Failed to mount procfs in WSL, continuing with existing /proc")
			} else {
				return fmt.Errorf("failed to mount proc filesystem: %v", err)
			}
		}
	}

	return nil
}

// setupNamespacesIndividually attempts to create namespaces one by one
// This is a fallback for WSL environments where combined unshare might fail
func setupNamespacesIndividually(hostname string, usePID, useIPC, useUTS, useNet, useMount bool) error {
	// Try each namespace individually
	if useUTS {
		if err := syscall.Unshare(NEWUTS); err != nil {
			log.Printf("WARNING: Failed to create UTS namespace: %v", err)
		} else if hostname != "" {
			if err := syscall.Sethostname([]byte(hostname)); err != nil {
				log.Printf("WARNING: Failed to set hostname: %v", err)
			}
		}
	}

	if useIPC {
		if err := syscall.Unshare(NEWIPC); err != nil {
			log.Printf("WARNING: Failed to create IPC namespace: %v", err)
		}
	}

	if usePID {
		if err := syscall.Unshare(NEWPID); err != nil {
			log.Printf("WARNING: Failed to create PID namespace: %v", err)
		} else {
			// Make sure /proc exists
			if err := os.MkdirAll("/proc", 0755); err != nil {
				log.Printf("WARNING: Failed to create /proc directory: %v", err)
			} else {
				// Mount proc filesystem
				if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
					log.Printf("WARNING: Failed to mount proc filesystem: %v", err)
				}
			}
		}
	}

	if useMount {
		if err := syscall.Unshare(NEWNS); err != nil {
			log.Printf("WARNING: Failed to create mount namespace: %v", err)
		}
	}

	if useNet {
		if err := syscall.Unshare(NEWNET); err != nil {
			log.Printf("WARNING: Failed to create network namespace: %v", err)
		}
	}

	return nil
}

// SetupUserNamespace sets up a user namespace with UID/GID mappings
func SetupUserNamespace(uid, gid int) error {
	// Create a new user namespace
	if err := syscall.Unshare(NEWUSER); err != nil {
		return fmt.Errorf("failed to unshare user namespace: %v", err)
	}

	// Write UID mapping
	uidMapPath := "/proc/self/uid_map"
	uidMapping := fmt.Sprintf("0 %d 1", uid)
	if err := os.WriteFile(uidMapPath, []byte(uidMapping), 0644); err != nil {
		return fmt.Errorf("failed to write UID mapping: %v", err)
	}

	// Set "deny" in /proc/self/setgroups before writing GID map
	if err := os.WriteFile("/proc/self/setgroups", []byte("deny"), 0644); err != nil {
		return fmt.Errorf("failed to set setgroups to deny: %v", err)
	}

	// Write GID mapping
	gidMapPath := "/proc/self/gid_map"
	gidMapping := fmt.Sprintf("0 %d 1", gid)
	if err := os.WriteFile(gidMapPath, []byte(gidMapping), 0644); err != nil {
		return fmt.Errorf("failed to write GID mapping: %v", err)
	}

	return nil
}

// JoinNamespace joins an existing namespace using its path
func JoinNamespace(nsType int, nsPath string) error {
	// Open the namespace file
	ns, err := os.Open(nsPath)
	if err != nil {
		return fmt.Errorf("failed to open namespace file: %v", err)
	}
	defer ns.Close()

	// Get file descriptor
	fd := ns.Fd()

	// Join the namespace
	const SYS_SETNS = 308
	_, _, errno := syscall.Syscall(SYS_SETNS, uintptr(fd), uintptr(nsType), 0)
	if errno != 0 {
		return fmt.Errorf("failed to join namespace: %v", errno)
	}
	return nil
}

// CreateNamespaceFile creates a bind mount of a namespace for later use
func CreateNamespaceFile(nsType int, nsPath string) error {
	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(nsPath), 0755); err != nil {
		return fmt.Errorf("failed to create namespace directory: %v", err)
	}

	// Determine the source namespace path based on type
	var nsName string
	switch nsType {
	case NEWUTS:
		nsName = "uts"
	case NEWIPC:
		nsName = "ipc"
	case NEWPID:
		nsName = "pid"
	case NEWNS:
		nsName = "mnt"
	case NEWNET:
		nsName = "net"
	case NEWUSER:
		nsName = "user"
	default:
		return fmt.Errorf("unknown namespace type: %d", nsType)
	}

	// Source is the current process's namespace
	source := fmt.Sprintf("/proc/self/ns/%s", nsName)

	// Create the target file if it doesn't exist
	f, err := os.Create(nsPath)
	if err != nil {
		return fmt.Errorf("failed to create namespace file: %v", err)
	}
	f.Close()

	// Bind mount the namespace to the file
	if err := syscall.Mount(source, nsPath, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to bind mount namespace: %v", err)
	}

	return nil
}

// SetupNetworkNamespace configures the network namespace with a loopback interface
func SetupNetworkNamespace() error {
	// This would typically use netlink to configure the network interface
	// For educational purposes, we'll use the ip command
	cmds := [][]string{
		{"ip", "link", "set", "lo", "up"},
	}

	for _, cmd := range cmds {
		if err := syscall.Exec(cmd[0], cmd, os.Environ()); err != nil {
			return fmt.Errorf("failed to execute %s: %v", cmd[0], err)
		}
	}

	return nil
}

// PivotRoot changes the root filesystem to a new location
func PivotRoot(newRoot string) error {
	if utils.IsWSL() {
		// In WSL, pivot_root can be problematic, so we use chroot as a fallback
		if worked, msg := utils.TryWSLCompat("pivot_root"); !worked {
			fmt.Printf("WARNING: %s\n", msg)
			fmt.Println("Using chroot instead of pivot_root in WSL environment")
			return ChrootFallback(newRoot)
		}
	}

	// Ensure the new root is mounted as MS_BIND
	if err := syscall.Mount(newRoot, newRoot, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("failed to bind mount rootfs: %v", err)
	}

	// Create temporary directory for old root
	pivotDir := filepath.Join(newRoot, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0700); err != nil {
		return fmt.Errorf("failed to create pivot_root dir: %v", err)
	}

	// Pivot the root
	if err := syscall.PivotRoot(newRoot, pivotDir); err != nil {
		// If pivot_root fails, try the chroot fallback
		if utils.IsWSL() {
			fmt.Printf("WARNING: pivot_root failed in WSL: %v\n", err)
			fmt.Println("Falling back to chroot method")
			return ChrootFallback(newRoot)
		}
		return fmt.Errorf("failed to pivot_root: %v", err)
	}

	// Change to the new root
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir to new root: %v", err)
	}

	// Unmount the old root and remove the temporary directory
	oldRoot := "/.pivot_root"
	if err := syscall.Unmount(oldRoot, syscall.MNT_DETACH); err != nil {
		fmt.Printf("WARNING: Could not unmount old root: %v\n", err)
		// Continue anyway as this might be acceptable in some environments
	}

	if err := os.RemoveAll(oldRoot); err != nil {
		fmt.Printf("WARNING: Could not remove pivot_root dir: %v\n", err)
		// Continue anyway as this might be acceptable in some environments
	}

	return nil
}

// ChrootFallback provides a chroot-based fallback for WSL environments
// where pivot_root might not work correctly
func ChrootFallback(newRoot string) error {
	// Ensure essential mount points exist in the new root
	mountPoints := []struct {
		source string
		target string
		fstype string
		flags  uintptr
		data   string
	}{
		{"proc", filepath.Join(newRoot, "proc"), "proc", 0, ""},
		{"sys", filepath.Join(newRoot, "sys"), "sysfs", 0, ""},
		{"tmp", filepath.Join(newRoot, "tmp"), "tmpfs", 0, ""},
		{"/dev", filepath.Join(newRoot, "dev"), "", syscall.MS_BIND, ""},
	}

	for _, m := range mountPoints {
		// Ensure the mount point exists
		os.MkdirAll(m.target, 0755)

		// Mount the filesystem
		if m.source != "" && m.fstype != "" {
			err := syscall.Mount(m.source, m.target, m.fstype, m.flags, m.data)
			if err != nil {
				fmt.Printf("WARNING: Failed to mount %s: %v\n", m.target, err)
				// Continue anyway, as some mounts might be optional
			}
		}
	}

	// Change root
	if err := syscall.Chroot(newRoot); err != nil {
		return fmt.Errorf("failed to chroot: %v", err)
	}

	// Change to the root directory
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("failed to chdir to new root: %v", err)
	}

	fmt.Println("INFO: Successfully changed root with chroot (WSL compatibility mode)")
	return nil
}
