# Container Runtime Implementation Plan

## 1. OCI Runtime Spec Compliance

### 1.1 Bundle Structure
```
bundle/
├── config.json     # Container configuration
├── state.json      # Runtime state (generated)
└── rootfs/         # Container filesystem
```

### 1.2 Config.json Schema Implementation
```go
type Spec struct {
    Version     string      `json:"ociVersion"`
    Process     Process     `json:"process"`
    Root        Root        `json:"root"`
    Hostname    string      `json:"hostname"`
    Mounts      []Mount     `json:"mounts"`
    Linux       Linux       `json:"linux"`
    Windows     Windows     `json:"windows,omitempty"`
}

type Process struct {
    Terminal bool     `json:"terminal"`
    ConsoleSize Box   `json:"consoleSize,omitempty"`
    User      User    `json:"user"`
    Args      []string `json:"args"`
    Env       []string `json:"env"`
    Cwd       string   `json:"cwd"`
    Capabilities *LinuxCapabilities `json:"capabilities,omitempty"`
    Rlimits   []Rlimit `json:"rlimits,omitempty"`
    NoNewPrivileges bool `json:"noNewPrivileges"`
    ApparmorProfile string `json:"apparmorProfile,omitempty"`
    OOMScoreAdj *int `json:"oomScoreAdj,omitempty"`
    SelinuxLabel string `json:"selinuxLabel,omitempty"`
}

type Linux struct {
    Namespaces []Namespace `json:"namespaces"`
    Devices    []Device    `json:"devices"`
    CgroupsPath string     `json:"cgroupsPath"`
    Resources  *Resources  `json:"resources,omitempty"`
    Seccomp    *Seccomp    `json:"seccomp,omitempty"`
    RootfsPropagation string `json:"rootfsPropagation,omitempty"`
    MaskedPaths []string   `json:"maskedPaths,omitempty"`
    ReadonlyPaths []string `json:"readonlyPaths,omitempty"`
    MountLabel  string     `json:"mountLabel,omitempty"`
    IntelRdt   *IntelRdt  `json:"intelRdt,omitempty"`
}
```

### 1.3 State Management
```go
type State struct {
    Version     string    `json:"ociVersion"`
    ID          string    `json:"id"`
    Status      string    `json:"status"`
    Pid         int       `json:"pid"`
    Bundle      string    `json:"bundle"`
    Annotations map[string]string `json:"annotations,omitempty"`
}
```

### 1.4 Runtime Lifecycle Commands
```go
// create command
func create(bundlePath, containerID string) error {
    // 1. Validate bundle structure
    // 2. Parse config.json
    // 3. Create container state
    // 4. Initialize namespaces
    // 5. Set up cgroups
    // 6. Write state.json
}

// start command  
func start(containerID string) error {
    // 1. Read state.json
    // 2. Execute containerized process
    // 3. Update status to "running"
}

// kill command
func kill(containerID string, signal syscall.Signal) error {
    // 1. Send signal to container process
    // 2. Handle graceful shutdown
}

// delete command
func delete(containerID string) error {
    // 1. Clean up namespaces
    // 2. Remove cgroups
    // 3. Delete state.json
    // 4. Clean up filesystem
}
```

### 1.5 Hook Execution
```go
type Hook struct {
    Path string   `json:"path"`
    Args []string `json:"args,omitempty"`
    Env  []string `json:"env,omitempty"`
    Timeout *int  `json:"timeout,omitempty"`
}

func executeHooks(hooks []Hook, state State) error {
    for _, hook := range hooks {
        cmd := exec.Command(hook.Path, hook.Args...)
        cmd.Env = append(os.Environ(), hook.Env...)
        
        if hook.Timeout != nil {
            ctx, cancel := context.WithTimeout(context.Background(), 
                time.Duration(*hook.Timeout)*time.Second)
            defer cancel()
            cmd = exec.CommandContext(ctx, hook.Path, hook.Args...)
        }
        
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("hook execution failed: %v", err)
        }
    }
    return nil
}
```

### 1.6 Bundle Validation
```go
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
    config, err := parseConfig(configPath)
    if err != nil {
        return fmt.Errorf("invalid config.json: %v", err)
    }
    
    // Validate OCI version
    if config.Version != "1.0.2" {
        return fmt.Errorf("unsupported OCI version: %s", config.Version)
    }
    
    return nil
}
```
