package marketplace

import (
	"testing"
)

func TestDeriveSource(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    string
		wantErr bool
	}{
		{
			name:    "GitHub HTTPS URL",
			repoURL: "https://github.com/feed-mob/claude-code-marketplace",
			want:    "feed-mob/claude-code-marketplace",
			wantErr: false,
		},
		{
			name:    "GitHub HTTPS URL with .git suffix",
			repoURL: "https://github.com/anthropics/claude-code.git",
			want:    "anthropics/claude-code",
			wantErr: false,
		},
		{
			name:    "GitHub HTTPS URL with trailing slash",
			repoURL: "https://github.com/anthropics/skills/",
			want:    "anthropics/skills",
			wantErr: false,
		},
		{
			name:    "GitLab HTTPS URL",
			repoURL: "https://gitlab.com/company/plugins",
			want:    "https://gitlab.com/company/plugins",
			wantErr: false,
		},
		{
			name:    "Codeberg HTTPS URL",
			repoURL: "https://codeberg.org/user/marketplace",
			want:    "https://codeberg.org/user/marketplace",
			wantErr: false,
		},
		{
			name:    "Generic Git hosting",
			repoURL: "https://git.example.com/org/repo",
			want:    "https://git.example.com/org/repo",
			wantErr: false,
		},
		{
			name:    "Empty URL",
			repoURL: "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "Invalid URL",
			repoURL: "not a valid url",
			want:    "",
			wantErr: true,
		},
		{
			name:    "GitHub URL with insufficient path segments",
			repoURL: "https://github.com/owner",
			want:    "",
			wantErr: true,
		},
		{
			name:    "GitHub URL with just slash",
			repoURL: "https://github.com/",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeriveSource(tt.repoURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveSource(%q) error = %v, wantErr %v", tt.repoURL, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeriveSource(%q) = %q, want %q", tt.repoURL, got, tt.want)
			}
		})
	}
}

func TestIsGitHubRepo(t *testing.T) {
	tests := []struct {
		name    string
		repoURL string
		want    bool
	}{
		{
			name:    "GitHub HTTPS",
			repoURL: "https://github.com/owner/repo",
			want:    true,
		},
		{
			name:    "GitHub HTTPS with .git",
			repoURL: "https://github.com/owner/repo.git",
			want:    true,
		},
		{
			name:    "GitLab HTTPS",
			repoURL: "https://gitlab.com/company/plugins",
			want:    false,
		},
		{
			name:    "Codeberg",
			repoURL: "https://codeberg.org/user/marketplace",
			want:    false,
		},
		{
			name:    "Invalid URL",
			repoURL: "not a url",
			want:    false,
		},
		{
			name:    "Empty URL",
			repoURL: "",
			want:    false,
		},
		{
			name:    "GitHub subdomain",
			repoURL: "https://raw.githubusercontent.com/owner/repo/main/file.json",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGitHubRepo(tt.repoURL)
			if got != tt.want {
				t.Errorf("IsGitHubRepo(%q) = %v, want %v", tt.repoURL, got, tt.want)
			}
		})
	}
}
