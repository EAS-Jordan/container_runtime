package fs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// OverlayMount represents an overlay mount configuration
type OverlayMount struct {
	// Lower layers (read-only)
	LowerDirs []string
	// Upper layer (read-write)
	UpperDir string
	// Work directory for OverlayFS
	WorkDir string
	// Mount point
	MountPoint string
}

// SetupOverlay creates and mounts an overlay filesystem
func SetupOverlay(config OverlayMount) error {
	// Ensure directories exist
	if err := os.MkdirAll(config.UpperDir, 0755); err != nil {
		return fmt.Errorf("failed to create upper dir: %v", err)
	}

	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work dir: %v", err)
	}

	if err := os.MkdirAll(config.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %v", err)
	}

	// Join lower directories with colon
	lowerDirs := strings.Join(config.LowerDirs, ":")

	// Create mount options
	options := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDirs, config.UpperDir, config.WorkDir)

	// Mount the overlay filesystem
	if err := syscall.Mount("overlay", config.MountPoint, "overlay", 0, options); err != nil {
		return fmt.Errorf("failed to mount overlay: %v", err)
	}

	return nil
}

// UnmountOverlay unmounts an overlay filesystem
func UnmountOverlay(mountPoint string) error {
	if err := syscall.Unmount(mountPoint, 0); err != nil {
		return fmt.Errorf("failed to unmount overlay: %v", err)
	}
	return nil
}

// ExtractImage extracts a container image to create layers
func ExtractImage(imagePath, targetDir string) ([]string, error) {
	// This is a simplified version. In a real implementation,
	// we would parse the image manifest and extract each layer.

	// Create target directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create target directory: %v", err)
	}

	// Extract the image (assuming it's a tarball)
	cmd := exec.Command("tar", "-xf", imagePath, "-C", targetDir)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to extract image: %v", err)
	}

	// In a real implementation, we would return the layers in the correct order
	// Here, we'll just return the target directory as the only layer
	return []string{targetDir}, nil
}

// CreateBundle creates an OCI bundle from layers
func CreateBundle(layers []string, bundlePath string) error {
	// Create bundle directory
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return fmt.Errorf("failed to create bundle directory: %v", err)
	}

	// Create rootfs directory
	rootfsPath := filepath.Join(bundlePath, "rootfs")
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		return fmt.Errorf("failed to create rootfs directory: %v", err)
	}

	// Setup temporary directories for overlay
	tempDir := filepath.Join(bundlePath, ".temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	upperDir := filepath.Join(tempDir, "upper")
	workDir := filepath.Join(tempDir, "work")

	// Mount overlay
	overlay := OverlayMount{
		LowerDirs:  layers,
		UpperDir:   upperDir,
		WorkDir:    workDir,
		MountPoint: rootfsPath,
	}

	if err := SetupOverlay(overlay); err != nil {
		return fmt.Errorf("failed to setup overlay: %v", err)
	}

	return nil
}

// CleanupBundle cleans up an OCI bundle
func CleanupBundle(bundlePath string) error {
	// Unmount rootfs
	rootfsPath := filepath.Join(bundlePath, "rootfs")
	if err := UnmountOverlay(rootfsPath); err != nil {
		// Ignore error if not mounted
		if !strings.Contains(err.Error(), "not mounted") {
			return fmt.Errorf("failed to unmount rootfs: %v", err)
		}
	}

	// Remove bundle directory
	if err := os.RemoveAll(bundlePath); err != nil {
		return fmt.Errorf("failed to remove bundle directory: %v", err)
	}

	return nil
}

// PrepareRootfs prepares the container rootfs with mount points and devices
func PrepareRootfs(rootfs string) error {
	// Create necessary directories
	dirs := []string{
		"/proc",
		"/sys",
		"/dev",
		"/dev/pts",
		"/dev/shm",
		"/tmp",
		"/etc",
		"/var/run",
	}

	for _, dir := range dirs {
		path := filepath.Join(rootfs, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", path, err)
		}
	}

	// Create device nodes
	// In a real implementation, we would create more devices
	// This is just a minimal set for illustration
	devices := []struct {
		path  string
		major uint32
		minor uint32
		mode  uint32
	}{
		{filepath.Join(rootfs, "/dev/null"), 1, 3, 0666},
		{filepath.Join(rootfs, "/dev/zero"), 1, 5, 0666},
		{filepath.Join(rootfs, "/dev/full"), 1, 7, 0666},
		{filepath.Join(rootfs, "/dev/tty"), 5, 0, 0666},
		{filepath.Join(rootfs, "/dev/random"), 1, 8, 0666},
		{filepath.Join(rootfs, "/dev/urandom"), 1, 9, 0666},
	}

	for _, dev := range devices {
		if err := syscall.Mknod(dev.path, syscall.S_IFCHR|dev.mode, int(dev.major<<8|dev.minor)); err != nil {
			return fmt.Errorf("failed to create device %s: %v", dev.path, err)
		}
	}

	return nil
}

// SetupMounts sets up required mounts for a container
func SetupMounts(rootfs string) error {
	// Define mount points
	mounts := []struct {
		source      string
		destination string
		fstype      string
		flags       uintptr
		data        string
	}{
		{"proc", filepath.Join(rootfs, "proc"), "proc", 0, ""},
		{"sysfs", filepath.Join(rootfs, "sys"), "sysfs", syscall.MS_RDONLY, ""},
		{"tmpfs", filepath.Join(rootfs, "dev"), "tmpfs", 0, "mode=755,size=65536k"},
		{"devpts", filepath.Join(rootfs, "dev/pts"), "devpts", 0, "newinstance,ptmxmode=0666,mode=0620,gid=5"},
		{"tmpfs", filepath.Join(rootfs, "dev/shm"), "tmpfs", 0, "mode=1777,size=65536k"},
	}

	for _, m := range mounts {
		if err := syscall.Mount(m.source, m.destination, m.fstype, m.flags, m.data); err != nil {
			return fmt.Errorf("failed to mount %s to %s: %v", m.source, m.destination, err)
		}
	}

	return nil
}

// SetupBindMount sets up a bind mount for a container
func SetupBindMount(hostPath, containerPath string, readonly bool) error {
	// Check if host path exists
	if _, err := os.Stat(hostPath); os.IsNotExist(err) {
		return fmt.Errorf("host path %s does not exist", hostPath)
	}

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(containerPath), 0755); err != nil {
		return fmt.Errorf("failed to create container path directory: %v", err)
	}

	// Create the mount point if it doesn't exist
	if _, err := os.Stat(containerPath); os.IsNotExist(err) {
		hostInfo, err := os.Stat(hostPath)
		if err != nil {
			return fmt.Errorf("failed to stat host path: %v", err)
		}

		if hostInfo.IsDir() {
			if err := os.MkdirAll(containerPath, 0755); err != nil {
				return fmt.Errorf("failed to create mount point directory: %v", err)
			}
		} else {
			f, err := os.Create(containerPath)
			if err != nil {
				return fmt.Errorf("failed to create mount point file: %v", err)
			}
			f.Close()
		}
	}

	// Perform bind mount
	if err := syscall.Mount(hostPath, containerPath, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to bind mount %s to %s: %v", hostPath, containerPath, err)
	}

	// If readonly, remount as readonly
	if readonly {
		if err := syscall.Mount("", containerPath, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY, ""); err != nil {
			return fmt.Errorf("failed to remount %s as readonly: %v", containerPath, err)
		}
	}

	return nil
}
