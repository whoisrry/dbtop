package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dbtop/config"
	"dbtop/monitor"
)

func main() {
	// Get configuration file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}
	configPath := filepath.Join(homeDir, ".dbtop")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Determine which database instance to monitor
	var instanceName string
	if len(os.Args) > 1 {
		instanceName = os.Args[1]
	} else {
		// If no instance specified and only one exists, use it as default
		if len(cfg.Instances) == 1 {
			for name := range cfg.Instances {
				instanceName = name
				break
			}
		} else {
			fmt.Println("Usage: dbtop [instance_name]")
			fmt.Println("Available instances:")
			for name := range cfg.Instances {
				fmt.Printf("  - %s\n", name)
			}
			os.Exit(1)
		}
	}

	// Get the specified instance
	instance, exists := cfg.Instances[instanceName]
	if !exists {
		fmt.Printf("Instance '%s' not found in configuration\n", instanceName)
		fmt.Println("Available instances:")
		for name := range cfg.Instances {
			fmt.Printf("  - %s\n", name)
		}
		os.Exit(1)
	}

	// Start monitoring
	fmt.Printf("Starting monitoring for instance: %s (%s)\n", instanceName, instance.Type)
	monitor.Start(instanceName, instance)
}
