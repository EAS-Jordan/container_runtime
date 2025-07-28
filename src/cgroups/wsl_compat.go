package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/container-runtime/core/utils"
)

// MockCgroupManager provides a simulated cgroup manager for WSL environments
// where full cgroup functionality might not be available
type MockCgroupManager struct {
	baseDir string
}

// NewMockCgroupManager creates a mock cgroup manager for WSL environments
func NewMockCgroupManager(baseDir string) (*MockCgroupManager, error) {
	// Create the base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mock cgroup directory: %v", err)
	}

	return &MockCgroupManager{
		baseDir: baseDir,
	}, nil
}

// CreateCgroup creates a mock cgroup for a container
func (m *MockCgroupManager) CreateCgroup(containerID string) (string, error) {
	cgroupPath := filepath.Join(m.baseDir, containerID)

	// Create container cgroup directory
	if err := os.MkdirAll(cgroupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create mock cgroup: %v", err)
	}

	// Create mock controller directories
	controllers := []string{"cpu", "memory", "pids", "blkio"}
	for _, controller := range controllers {
		controllerPath := filepath.Join(cgroupPath, controller)
		if err := os.MkdirAll(controllerPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create mock controller directory: %v", err)
		}
	}

	fmt.Printf("INFO: Created mock cgroup at %s\n", cgroupPath)
	return cgroupPath, nil
}

// ApplyResourceLimits applies mock resource limits
func (m *MockCgroupManager) ApplyResourceLimits(cgroupPath string, limits ResourceLimits) error {
	// Write mock limit files
	if limits.MemoryLimit > 0 {
		memPath := filepath.Join(cgroupPath, "memory", "memory.limit_in_bytes")
		err := os.MkdirAll(filepath.Dir(memPath), 0755)
		if err == nil {
			err = os.WriteFile(memPath, []byte(strconv.FormatInt(limits.MemoryLimit, 10)), 0644)
		}
		if err != nil {
			fmt.Printf("WARNING: Failed to set mock memory limit: %v\n", err)
		}
	}

	if limits.CPUShares > 0 {
		cpuPath := filepath.Join(cgroupPath, "cpu", "cpu.shares")
		err := os.MkdirAll(filepath.Dir(cpuPath), 0755)
		if err == nil {
			err = os.WriteFile(cpuPath, []byte(strconv.FormatUint(limits.CPUShares, 10)), 0644)
		}
		if err != nil {
			fmt.Printf("WARNING: Failed to set mock CPU shares: %v\n", err)
		}
	}

	if limits.PidsMax > 0 {
		pidsPath := filepath.Join(cgroupPath, "pids", "pids.max")
		err := os.MkdirAll(filepath.Dir(pidsPath), 0755)
		if err == nil {
			err = os.WriteFile(pidsPath, []byte(strconv.FormatInt(limits.PidsMax, 10)), 0644)
		}
		if err != nil {
			fmt.Printf("WARNING: Failed to set mock PIDs limit: %v\n", err)
		}
	}

	fmt.Println("NOTE: Resource limits are simulated and not enforced in WSL")
	return nil
}

// AddProcess adds a process to the mock cgroup (simulation only)
func (m *MockCgroupManager) AddProcess(cgroupPath string, pid int) error {
	// Add process ID to mock cgroup.procs files
	controllers := []string{"cpu", "memory", "pids", "blkio"}
	for _, controller := range controllers {
		procsPath := filepath.Join(cgroupPath, controller, "cgroup.procs")
		err := os.MkdirAll(filepath.Dir(procsPath), 0755)
		if err == nil {
			err = os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644)
		}
		if err != nil {
			fmt.Printf("WARNING: Failed to add process to mock %s cgroup: %v\n", controller, err)
		}
	}

	fmt.Printf("NOTE: Process %d added to mock cgroups (simulation only)\n", pid)
	return nil
}

// RemoveCgroup removes a mock cgroup
func (m *MockCgroupManager) RemoveCgroup(cgroupPath string) error {
	if err := os.RemoveAll(cgroupPath); err != nil {
		return fmt.Errorf("failed to remove mock cgroup: %v", err)
	}
	return nil
}

// Detect if we're running in WSL with limited cgroup support
func detectWSLCgroupLimitations() bool {
	if !utils.IsWSL() {
		return false
	}

	// Check for cgroup v2 unified hierarchy
	_, err1 := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	if err1 == nil {
		// Has cgroup v2
		return false
	}

	// Check for cgroup v1 controllers
	_, err2 := os.Stat("/sys/fs/cgroup/memory")
	if err2 == nil {
		// Has cgroup v1
		return false
	}

	// No cgroup controllers found
	return true
}

// CreateMockCgroupIfNeeded creates a mock cgroup manager if real cgroups are not available
func CreateMockCgroupIfNeeded() (interface{}, error) {
	if detectWSLCgroupLimitations() {
		fmt.Println("WARNING: Limited cgroup support detected in WSL")
		fmt.Println("WARNING: Using mock cgroup implementation")
		return NewMockCgroupManager("/tmp/mock_cgroups")
	}
	return NewCgroupManager()
}
