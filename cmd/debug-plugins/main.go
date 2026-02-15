package main

import (
	"fmt"
	"log"

	"github.com/itsdevcoffee/plum/internal/config"
)

func main() {
	plugins, err := config.LoadAllPlugins()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total plugins loaded: %d\n\n", len(plugins))

	// Group by marketplace
	byMarketplace := make(map[string][]string)
	for _, p := range plugins {
		byMarketplace[p.Marketplace] = append(byMarketplace[p.Marketplace], p.Name)
	}

	for marketplace, pluginNames := range byMarketplace {
		fmt.Printf("%s (%d plugins):\n", marketplace, len(pluginNames))
		for _, name := range pluginNames {
			fmt.Printf("  - %s\n", name)
		}
		fmt.Println()
	}

	// Check for video-analysis specifically
	found := false
	for _, p := range plugins {
		if p.Name == "video-analysis" {
			fmt.Printf("✓ Found video-analysis@%s (Installed: %v)\n", p.Marketplace, p.Installed)
			found = true
		}
	}
	if !found {
		fmt.Println("✗ video-analysis NOT found in loaded plugins")
	}
}
