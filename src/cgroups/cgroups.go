package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ResourceLimits defines the resource constraints for a container
type ResourceLimits struct {
	// Memory limits in bytes
	MemoryLimit     int64
	MemorySoftLimit int64
	MemorySwapLimit int64

	// CPU limits
	CPUShares  uint64 // Relative share (1024 is default)
	CPUPeriod  uint64 // Period in microseconds
	CPUQuota   uint64 // Quota in microseconds
	CPUSetCPUs string // CPUs to use (e.g., "0-3,5")

	// PID limits
	PidsMax int64 // Maximum number of PIDs
}

// CgroupManager manages cgroup operations
type CgroupManager struct {
	// Path to the cgroup mount point
	cgroupRoot string
	// Whether to use cgroup v2
	isV2 bool
}

// NewCgroupManager creates a new cgroup manager
func NewCgroupManager() (*CgroupManager, error) {
	// Import utils package
	isWSL := false
	_, wslErr := os.ReadFile("/proc/version")
	if wslErr == nil {
		content, _ := os.ReadFile("/proc/version")
		isWSL = strings.Contains(strings.ToLower(string(content)), "microsoft")
	}

	// Check if cgroup v2 unified hierarchy is in use
	_, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	isV2 := err == nil

	// In WSL, check both v1 and v2 cgroups availability
	var cgroupRoot string
	if isV2 {
		cgroupRoot = "/sys/fs/cgroup"
		if isWSL {
			fmt.Println("INFO: Using cgroups v2 in WSL - some functionality may be limited")
		}
	} else {
		// Check if cgroup v1 controllers are available
		_, err := os.Stat("/sys/fs/cgroup/memory")
		hasCgroupV1 := err == nil

		if !hasCgroupV1 && isWSL {
			fmt.Println("WARNING: No cgroup controllers found in WSL")
			fmt.Println("WARNING: Resource limits will be simulated (not enforced)")

			// Create a fallback directory for our mock cgroups
			mockDir := "/tmp/mock_cgroups"
			os.MkdirAll(mockDir, 0755)
			cgroupRoot = mockDir
		} else {
			cgroupRoot = "/sys/fs/cgroup"
			if isWSL {
				fmt.Println("INFO: Using cgroups v1 in WSL - some functionality may be limited")
			}
		}
	}

	return &CgroupManager{
		cgroupRoot: cgroupRoot,
		isV2:       isV2,
	}, nil
}

// CreateCgroup creates a new cgroup for a container
func (m *CgroupManager) CreateCgroup(containerID string) (string, error) {
	var cgroupPath string

	if m.isV2 {
		// For cgroup v2, create the container cgroup directly under the root
		cgroupPath = filepath.Join(m.cgroupRoot, containerID)
	} else {
		// For cgroup v1, we need to create a cgroup in each controller directory
		cgroupPath = containerID
		controllers := []string{"cpu", "cpuset", "memory", "pids", "blkio"}

		for _, controller := range controllers {
			controllerPath := filepath.Join(m.cgroupRoot, controller, cgroupPath)
			if err := os.MkdirAll(controllerPath, 0755); err != nil {
				return "", fmt.Errorf("failed to create cgroup %s: %v", controllerPath, err)
			}
		}
	}

	// For cgroup v2, create the container cgroup
	if m.isV2 {
		if err := os.MkdirAll(cgroupPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create cgroup %s: %v", cgroupPath, err)
		}

		// Enable controllers in the parent directory
		controllersPath := filepath.Join(m.cgroupRoot, "cgroup.subtree_control")
		if _, err := os.Stat(controllersPath); err == nil {
			// Read available controllers
			data, err := os.ReadFile(filepath.Join(m.cgroupRoot, "cgroup.controllers"))
			if err != nil {
				return "", fmt.Errorf("failed to read controllers: %v", err)
			}

			controllers := strings.Fields(string(data))
			var enables []string
			for _, c := range controllers {
				enables = append(enables, "+"+c)
			}

			// Enable controllers in parent
			if err := os.WriteFile(controllersPath, []byte(strings.Join(enables, " ")), 0644); err != nil {
				return "", fmt.Errorf("failed to enable controllers: %v", err)
			}
		}
	}

	return cgroupPath, nil
}

// ApplyResourceLimits applies resource limits to a cgroup
func (m *CgroupManager) ApplyResourceLimits(cgroupPath string, limits ResourceLimits) error {
	if m.isV2 {
		return m.applyV2Limits(cgroupPath, limits)
	}
	return m.applyV1Limits(cgroupPath, limits)
}

// applyV2Limits applies resource limits using cgroup v2
func (m *CgroupManager) applyV2Limits(cgroupPath string, limits ResourceLimits) error {
	// Apply memory limits
	if limits.MemoryLimit > 0 {
		memMax := filepath.Join(cgroupPath, "memory.max")
		if err := os.WriteFile(memMax, []byte(strconv.FormatInt(limits.MemoryLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory limit: %v", err)
		}
	}

	if limits.MemorySoftLimit > 0 {
		memHigh := filepath.Join(cgroupPath, "memory.high")
		if err := os.WriteFile(memHigh, []byte(strconv.FormatInt(limits.MemorySoftLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory soft limit: %v", err)
		}
	}

	if limits.MemorySwapLimit > 0 {
		memSwapMax := filepath.Join(cgroupPath, "memory.swap.max")
		if err := os.WriteFile(memSwapMax, []byte(strconv.FormatInt(limits.MemorySwapLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory swap limit: %v", err)
		}
	}

	// Apply CPU limits
	if limits.CPUQuota > 0 && limits.CPUPeriod > 0 {
		cpuMax := filepath.Join(cgroupPath, "cpu.max")
		value := fmt.Sprintf("%d %d", limits.CPUQuota, limits.CPUPeriod)
		if err := os.WriteFile(cpuMax, []byte(value), 0644); err != nil {
			return fmt.Errorf("failed to set CPU quota/period: %v", err)
		}
	}

	if limits.CPUShares > 0 {
		cpuWeight := filepath.Join(cgroupPath, "cpu.weight")
		// Convert from shares (CFS) to weight (cgroup v2)
		// 1-10000 range for weight, with 100 as default
		// 2-262144 range for shares, with 1024 as default
		weight := 1 + ((limits.CPUShares-2)*9999)/262142
		if err := os.WriteFile(cpuWeight, []byte(strconv.FormatUint(weight, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set CPU weight: %v", err)
		}
	}

	// Apply CPU set constraints
	if limits.CPUSetCPUs != "" {
		cpusetCpus := filepath.Join(cgroupPath, "cpuset.cpus")
		if err := os.WriteFile(cpusetCpus, []byte(limits.CPUSetCPUs), 0644); err != nil {
			return fmt.Errorf("failed to set cpuset.cpus: %v", err)
		}
	}

	// Apply PID limits
	if limits.PidsMax > 0 {
		pidsMax := filepath.Join(cgroupPath, "pids.max")
		if err := os.WriteFile(pidsMax, []byte(strconv.FormatInt(limits.PidsMax, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set pids.max: %v", err)
		}
	}

	return nil
}

// applyV1Limits applies resource limits using cgroup v1
func (m *CgroupManager) applyV1Limits(cgroupPath string, limits ResourceLimits) error {
	// Apply memory limits
	if limits.MemoryLimit > 0 {
		memLimit := filepath.Join(m.cgroupRoot, "memory", cgroupPath, "memory.limit_in_bytes")
		if err := os.WriteFile(memLimit, []byte(strconv.FormatInt(limits.MemoryLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory limit: %v", err)
		}
	}

	if limits.MemorySoftLimit > 0 {
		memSoftLimit := filepath.Join(m.cgroupRoot, "memory", cgroupPath, "memory.soft_limit_in_bytes")
		if err := os.WriteFile(memSoftLimit, []byte(strconv.FormatInt(limits.MemorySoftLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory soft limit: %v", err)
		}
	}

	if limits.MemorySwapLimit > 0 {
		memSwapLimit := filepath.Join(m.cgroupRoot, "memory", cgroupPath, "memory.memsw.limit_in_bytes")
		if err := os.WriteFile(memSwapLimit, []byte(strconv.FormatInt(limits.MemorySwapLimit, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set memory swap limit: %v", err)
		}
	}

	// Apply CPU limits
	if limits.CPUShares > 0 {
		cpuShares := filepath.Join(m.cgroupRoot, "cpu", cgroupPath, "cpu.shares")
		if err := os.WriteFile(cpuShares, []byte(strconv.FormatUint(limits.CPUShares, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set cpu.shares: %v", err)
		}
	}

	if limits.CPUQuota > 0 {
		cpuQuota := filepath.Join(m.cgroupRoot, "cpu", cgroupPath, "cpu.cfs_quota_us")
		if err := os.WriteFile(cpuQuota, []byte(strconv.FormatUint(limits.CPUQuota, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set cpu.cfs_quota_us: %v", err)
		}
	}

	if limits.CPUPeriod > 0 {
		cpuPeriod := filepath.Join(m.cgroupRoot, "cpu", cgroupPath, "cpu.cfs_period_us")
		if err := os.WriteFile(cpuPeriod, []byte(strconv.FormatUint(limits.CPUPeriod, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set cpu.cfs_period_us: %v", err)
		}
	}

	// Apply CPU set constraints
	if limits.CPUSetCPUs != "" {
		cpusetCpus := filepath.Join(m.cgroupRoot, "cpuset", cgroupPath, "cpuset.cpus")
		if err := os.WriteFile(cpusetCpus, []byte(limits.CPUSetCPUs), 0644); err != nil {
			return fmt.Errorf("failed to set cpuset.cpus: %v", err)
		}

		// Ensure cpuset.mems is set (required for cpuset controller)
		cpusetMems := filepath.Join(m.cgroupRoot, "cpuset", cgroupPath, "cpuset.mems")
		parentMems := filepath.Join(m.cgroupRoot, "cpuset", "cpuset.mems")

		data, err := os.ReadFile(parentMems)
		if err != nil {
			return fmt.Errorf("failed to read cpuset.mems: %v", err)
		}

		if err := os.WriteFile(cpusetMems, data, 0644); err != nil {
			return fmt.Errorf("failed to set cpuset.mems: %v", err)
		}
	}

	// Apply PID limits
	if limits.PidsMax > 0 {
		pidsMax := filepath.Join(m.cgroupRoot, "pids", cgroupPath, "pids.max")
		if err := os.WriteFile(pidsMax, []byte(strconv.FormatInt(limits.PidsMax, 10)), 0644); err != nil {
			return fmt.Errorf("failed to set pids.max: %v", err)
		}
	}

	return nil
}

// AddProcess adds a process to the cgroup
func (m *CgroupManager) AddProcess(cgroupPath string, pid int) error {
	if m.isV2 {
		// For cgroup v2, add process to the cgroup.procs file
		procsPath := filepath.Join(m.cgroupRoot, cgroupPath, "cgroup.procs")
		if err := os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
			return fmt.Errorf("failed to add process to cgroup: %v", err)
		}
	} else {
		// For cgroup v1, add the process to each controller
		controllers := []string{"cpu", "cpuset", "memory", "pids", "blkio"}
		for _, controller := range controllers {
			procsPath := filepath.Join(m.cgroupRoot, controller, cgroupPath, "cgroup.procs")
			if err := os.WriteFile(procsPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
				return fmt.Errorf("failed to add process to %s cgroup: %v", controller, err)
			}
		}
	}

	return nil
}

// RemoveCgroup removes a cgroup
func (m *CgroupManager) RemoveCgroup(cgroupPath string) error {
	if m.isV2 {
		// For cgroup v2, remove the cgroup directory
		if err := os.RemoveAll(filepath.Join(m.cgroupRoot, cgroupPath)); err != nil {
			return fmt.Errorf("failed to remove cgroup: %v", err)
		}
	} else {
		// For cgroup v1, remove the cgroup from each controller
		controllers := []string{"cpu", "cpuset", "memory", "pids", "blkio"}
		for _, controller := range controllers {
			controllerPath := filepath.Join(m.cgroupRoot, controller, cgroupPath)
			if err := os.RemoveAll(controllerPath); err != nil {
				return fmt.Errorf("failed to remove %s cgroup: %v", controller, err)
			}
		}
	}

	return nil
}
