package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/container-runtime/core/runtime"
)

func main() {
	// Define command-line flags
	var (
		bundlePath  string
		rootPath    string
		containerID string
		action      string
	)

	// Parse flags
	flag.StringVar(&bundlePath, "bundle", "", "Path to the OCI bundle directory")
	flag.StringVar(&rootPath, "root", "/var/run/container", "Path to the runtime state directory")
	flag.StringVar(&containerID, "id", "", "Container ID")
	flag.Parse()

	// Determine the action from the first non-flag argument
	args := flag.Args()
	if len(args) > 0 {
		action = args[0]
	}

	//Debug: print the detected action
	fmt.Printf("DEBUG: Action detected: '%s'\n", action)
	fmt.Printf("DEBUG: Bundle path: '%s'\n", bundlePath)
	fmt.Printf("DEBUG: Action == 'create': %t\n", action == "create")
	fmt.Printf("DEBUG: BundlePath == '': %t\n", bundlePath == "")

	// Ensure required flags are provided
	if action == "create" && bundlePath == "" {
		fmt.Println("Bundle path is required for create action")
		os.Exit(1)
	}

	if containerID == "" {
		fmt.Println("Container ID is required")
		os.Exit(1)
	}

	// Execute the requested action
	switch action {
	case "create":
		if err := runtime.CreateContainer(bundlePath, rootPath, containerID); err != nil {
			fmt.Printf("Failed to create container: %v\n", err)
			os.Exit(1)
		}
	case "start":
		if err := runtime.StartContainer(rootPath, containerID); err != nil {
			fmt.Printf("Failed to start container: %v\n", err)
			os.Exit(1)
		}
	case "kill":
		signal := "SIGTERM"
		if len(args) > 1 {
			signal = args[1]
		}
		if err := runtime.KillContainer(rootPath, containerID, signal); err != nil {
			fmt.Printf("Failed to kill container: %v\n", err)
			os.Exit(1)
		}
	case "delete":
		if err := runtime.DeleteContainer(rootPath, containerID); err != nil {
			fmt.Printf("Failed to delete container: %v\n", err)
			os.Exit(1)
		}
	case "state":
		state, err := runtime.ContainerState(rootPath, containerID)
		if err != nil {
			fmt.Printf("Failed to get container state: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(state)
	default:
		fmt.Printf("Unknown action: %s\n", action)
		fmt.Println("Available actions: create, start, kill, delete, state")
		os.Exit(1)
	}
}
