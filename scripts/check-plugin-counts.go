package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MarketplaceManifest struct {
	Name    string                 `json:"name"`
	Plugins []map[string]interface{} `json:"plugins"`
}

var marketplaces = map[string]string{
	"claude-code-plugins-plus":   "jeremylongshore/claude-code-plugins-plus-skills",
	"claude-code-marketplace":    "ananddtyagi/cc-marketplace",
	"claude-code-plugins":        "anthropics/claude-code",
	"mag-claude-plugins":         "MadAppGang/claude-code",
	"dev-gom-plugins":            "Dev-GOM/claude-code-marketplace",
	"feedmob-claude-plugins":     "feed-mob/claude-code-marketplace",
	"claude-plugins-official":    "anthropics/claude-plugins-official",
	"anthropic-agent-skills":     "anthropics/skills",
	"wshobson-agents":            "wshobson/agents",
	"docker-plugins":             "docker/claude-plugins",
	"ccplugins-marketplace":      "ccplugins/marketplace",
	"claude-mem":                 "thedotmack/claude-mem",
}

func fetchPluginCount(repo string) (int, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/main/.claude-plugin/marketplace.json", repo)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var manifest MarketplaceManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return 0, err
	}

	return len(manifest.Plugins), nil
}

func main() {
	fmt.Println("🔍 Checking plugin counts from all marketplaces...")
	fmt.Println("Date:", time.Now().Format("2006-01-02"))
	fmt.Println()
	fmt.Println("Marketplace                    | Plugins | Status")
	fmt.Println("-------------------------------|---------|--------")

	for name, repo := range marketplaces {
		count, err := fetchPluginCount(repo)
		if err != nil {
			fmt.Printf("%-30s | %-7s | ❌ Error: %v\n", name, "N/A", err)
			continue
		}
		fmt.Printf("%-30s | %-7d | ✅\n", name, count)
		time.Sleep(500 * time.Millisecond) // Rate limit friendly
	}

	fmt.Println()
	fmt.Println("💡 Update discovery.go descriptions and README.md with accurate counts")
}
