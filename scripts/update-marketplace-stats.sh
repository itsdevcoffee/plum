#!/bin/bash
#
# Update Marketplace GitHub Stats
#
# This script fetches fresh GitHub stats for all marketplaces and displays
# them in a format ready to update internal/marketplace/discovery.go
#
# Usage: ./scripts/update-marketplace-stats.sh
#

set -e

echo "🔍 Fetching fresh GitHub stats for all Plum marketplaces..."
echo "Snapshot date: $(date -u +%Y-%m-%d)"
echo ""
echo "Format: StaticStats: &GitHubStats{Stars: X, Forks: Y, LastPushedAt: mustParseTime(\"...\"), OpenIssues: Z}"
echo ""
echo "=================================================="
echo ""

# Marketplace repos (from PopularMarketplaces)
declare -A repos=(
  ["claude-code-plugins-plus"]="jeremylongshore/claude-code-plugins-plus-skills"
  ["claude-code-marketplace"]="ananddtyagi/cc-marketplace"
  ["claude-code-plugins"]="anthropics/claude-code"
  ["mag-claude-plugins"]="MadAppGang/claude-code"
  ["dev-gom-plugins"]="Dev-GOM/claude-code-marketplace"
  ["feedmob-claude-plugins"]="feed-mob/claude-code-marketplace"
  ["claude-plugins-official"]="anthropics/claude-plugins-official"
  ["anthropic-agent-skills"]="anthropics/skills"
  ["wshobson-agents"]="wshobson/agents"
  ["docker-plugins"]="docker/claude-plugins"
  ["ccplugins-marketplace"]="ccplugins/marketplace"
  ["claude-mem"]="thedotmack/claude-mem"
)

for name in "${!repos[@]}"; do
  repo="${repos[$name]}"

  echo "=== $name ==="
  echo "Repo: https://github.com/$repo"

  response=$(curl -s -H "Accept: application/vnd.github.v3+json" \
    "https://api.github.com/repos/$repo")

  # Check for errors
  error_msg=$(echo "$response" | jq -r '.message // empty')
  if [ -n "$error_msg" ]; then
    echo "❌ ERROR: $error_msg"
    echo ""
    continue
  fi

  # Extract stats
  stars=$(echo "$response" | jq -r '.stargazers_count')
  forks=$(echo "$response" | jq -r '.forks_count')
  pushed=$(echo "$response" | jq -r '.pushed_at')
  issues=$(echo "$response" | jq -r '.open_issues_count')

  # Format for Go code
  echo "✅ StaticStats: &GitHubStats{"
  echo "    Stars:        $stars,"
  echo "    Forks:        $forks,"
  echo "    LastPushedAt: mustParseTime(\"$pushed\"),"
  echo "    OpenIssues:   $issues,"
  echo "},"
  echo ""

  # Rate limit friendly
  sleep 1
done

echo "=================================================="
echo ""
echo "📝 Next steps:"
echo "1. Update internal/marketplace/discovery.go with the stats above"
echo "2. Update the snapshot date comment"
echo "3. Run: go test ./internal/marketplace"
echo "4. Run: go build -o ./plum ./cmd/plum"
echo "5. Test marketplace browser for stats display"
echo ""
echo "💡 Tip: Save this output and compare with current discovery.go"
