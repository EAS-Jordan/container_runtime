package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

// Spec represents the OCI runtime specification
type Spec struct {
	Version  string  `json:"ociVersion"`
	Process  Process `json:"process"`
	Root     Root    `json:"root"`
	Hostname string  `json:"hostname"`
	Mounts   []Mount `json:"mounts"`
	Linux    Linux   `json:"linux"`
}

type Process struct {
	Terminal        bool               `json:"terminal"`
	User            User               `json:"user"`
	Args            []string           `json:"args"`
	Env             []string           `json:"env"`
	Cwd             string             `json:"cwd"`
	Capabilities    *LinuxCapabilities `json:"capabilities,omitempty"`
	NoNewPrivileges bool               `json:"noNewPrivileges"`
}

type User struct {
	UID uint32 `json:"uid"`
	GID uint32 `json:"gid"`
}

type LinuxCapabilities struct {
	Bounding    []string `json:"bounding"`
	Effective   []string `json:"effective"`
	Inheritable []string `json:"inheritable"`
	Permitted   []string `json:"permitted"`
	Ambient     []string `json:"ambient"`
}

type Root struct {
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
}

type Mount struct {
	Destination string   `json:"destination"`
	Type        string   `json:"type"`
	Source      string   `json:"source"`
	Options     []string `json:"options"`
}

type Linux struct {
	Namespaces    []Namespace   `json:"namespaces"`
	Resources     *Resources    `json:"resources,omitempty"`
	Devices       []LinuxDevice `json:"devices"`
	MaskedPaths   []string      `json:"maskedPaths,omitempty"`
	ReadonlyPaths []string      `json:"readonlyPaths,omitempty"`
}

type Namespace struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

type Resources struct {
	Memory *MemoryResources `json:"memory,omitempty"`
	CPU    *CPUResources    `json:"cpu,omitempty"`
	Pids   *PidsResources   `json:"pids,omitempty"`
}

type MemoryResources struct {
	Limit int64 `json:"limit,omitempty"`
}

type CPUResources struct {
	Shares int64  `json:"shares,omitempty"`
	Quota  int64  `json:"quota,omitempty"`
	Period int64  `json:"period,omitempty"`
	Cpus   string `json:"cpus,omitempty"`
}

type PidsResources struct {
	Limit int64 `json:"limit,omitempty"`
}

type LinuxDevice struct {
	Allow  bool   `json:"allow"`
	Access string `json:"access"`
}

// State represents the container state
type State struct {
	Version     string            `json:"ociVersion"`
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Pid         int               `json:"pid"`
	Bundle      string            `json:"bundle"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CreateContainer creates a new container
func CreateContainer(bundlePath, rootPath, containerID string) error {
	// Validate bundle
	if err := validateBundle(bundlePath); err != nil {
		return err
	}

	// Create container directory
	containerDir := filepath.Join(rootPath, containerID)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("failed to create container directory: %v", err)
	}

	// Read the config.json
	configPath := filepath.Join(bundlePath, "config.json")
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.json: %v", err)
	}

	var spec Spec
	if err := json.Unmarshal(configData, &spec); err != nil {
		return fmt.Errorf("failed to parse config.json: %v", err)
	}

	// Create initial state
	state := State{
		Version:     spec.Version,
		ID:          containerID,
		Status:      "created",
		Bundle:      bundlePath,
		Annotations: make(map[string]string),
	}

	// Write state.json
	stateData, err := json.MarshalIndent(state, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	statePath := filepath.Join(containerDir, "state.json")
	if err := ioutil.WriteFile(statePath, stateData, 0644); err != nil {
		return fmt.Errorf("failed to write state.json: %v", err)
	}

	return nil
}

// StartContainer starts a created container
func StartContainer(rootPath, containerID string) error {
	// Read the container state
	containerDir := filepath.Join(rootPath, containerID)
	statePath := filepath.Join(containerDir, "state.json")

	stateData, err := ioutil.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state.json: %v", err)
	}

	var state State
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to parse state.json: %v", err)
	}

	if state.Status != "created" {
		return fmt.Errorf("container is not in created state")
	}

	// Read the config.json
	configPath := filepath.Join(state.Bundle, "config.json")
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.json: %v", err)
	}

	var spec Spec
	if err := json.Unmarshal(configData, &spec); err != nil {
		return fmt.Errorf("failed to parse config.json: %v", err)
	}

	// Execute the runc binary to start the container
	// In a real implementation, we'd implement the container runtime ourselves
	// For educational purposes, we'll use runc as a reference
	cmd := exec.Command("runc", "run", containerID)
	cmd.Dir = state.Bundle
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start container: %v", err)
	}

	// Update state
	state.Status = "running"
	state.Pid = cmd.Process.Pid

	// Write updated state
	stateData, err = json.MarshalIndent(state, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %v", err)
	}

	if err := ioutil.WriteFile(statePath, stateData, 0644); err != nil {
		return fmt.Errorf("failed to write state.json: %v", err)
	}

	return nil
}

// KillContainer sends a signal to the container
func KillContainer(rootPath, containerID, signal string) error {
	// Read the container state
	containerDir := filepath.Join(rootPath, containerID)
	statePath := filepath.Join(containerDir, "state.json")

	stateData, err := ioutil.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state.json: %v", err)
	}

	var state State
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to parse state.json: %v", err)
	}

	if state.Status != "running" {
		return fmt.Errorf("container is not running")
	}

	// Convert signal string to syscall.Signal
	var sigNum syscall.Signal
	switch signal {
	case "SIGKILL", "KILL":
		sigNum = syscall.SIGKILL
	case "SIGTERM", "TERM":
		sigNum = syscall.SIGTERM
	case "SIGINT", "INT":
		sigNum = syscall.SIGINT
	default:
		// Try to parse as a number
		if s, err := strconv.Atoi(signal); err == nil {
			sigNum = syscall.Signal(s)
		} else {
			return fmt.Errorf("unknown signal: %s", signal)
		}
	}

	// Send signal to process
	process, err := os.FindProcess(state.Pid)
	if err != nil {
		return fmt.Errorf("failed to find process: %v", err)
	}

	if err := process.Signal(sigNum); err != nil {
		return fmt.Errorf("failed to send signal: %v", err)
	}

	return nil
}

// DeleteContainer deletes a container
func DeleteContainer(rootPath, containerID string) error {
	// Read the container state
	containerDir := filepath.Join(rootPath, containerID)
	statePath := filepath.Join(containerDir, "state.json")

	stateData, err := ioutil.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state.json: %v", err)
	}

	var state State
	if err := json.Unmarshal(stateData, &state); err != nil {
		return fmt.Errorf("failed to parse state.json: %v", err)
	}

	if state.Status == "running" {
		return fmt.Errorf("container is still running")
	}

	// Delete container directory
	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("failed to delete container directory: %v", err)
	}

	return nil
}

// ContainerState returns the state of a container
func ContainerState(rootPath, containerID string) (string, error) {
	// Read the container state
	containerDir := filepath.Join(rootPath, containerID)
	statePath := filepath.Join(containerDir, "state.json")

	stateData, err := ioutil.ReadFile(statePath)
	if err != nil {
		return "", fmt.Errorf("failed to read state.json: %v", err)
	}

	var state State
	if err := json.Unmarshal(stateData, &state); err != nil {
		return "", fmt.Errorf("failed to parse state.json: %v", err)
	}

	// Update state if container is running
	if state.Status == "running" {
		// Check if process is still running
		process, err := os.FindProcess(state.Pid)
		if err != nil || process.Signal(syscall.Signal(0)) != nil {
			// Process is not running
			state.Status = "stopped"

			// Write updated state
			stateData, err = json.MarshalIndent(state, "", "    ")
			if err != nil {
				return "", fmt.Errorf("failed to marshal state: %v", err)
			}

			if err := ioutil.WriteFile(statePath, stateData, 0644); err != nil {
				return "", fmt.Errorf("failed to write state.json: %v", err)
			}
		}
	}

	return string(stateData), nil
}

// validateBundle validates the bundle structure
func validateBundle(bundlePath string) error {
	// Check required files exist
	configPath := filepath.Join(bundlePath, "config.json")
	rootfsPath := filepath.Join(bundlePath, "rootfs")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config.json not found")
	}

	if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
		return fmt.Errorf("rootfs not found")
	}

	// Parse and validate config.json
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config.json: %v", err)
	}

	var spec Spec
	if err := json.Unmarshal(configData, &spec); err != nil {
		return fmt.Errorf("invalid config.json: %v", err)
	}

	// Validate OCI version
	if spec.Version != "1.0.2" {
		return fmt.Errorf("unsupported OCI version: %s", spec.Version)
	}

	return nil
}
