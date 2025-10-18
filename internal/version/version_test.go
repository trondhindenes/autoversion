package version

import (
	"testing"
)

func TestVersionString(t *testing.T) {
	tests := []struct {
		name     string
		version  Version
		expected string
	}{
		{
			name: "main branch version",
			version: Version{
				Major: 1,
				Minor: 0,
				Patch: 0,
			},
			expected: "1.0.0",
		},
		{
			name: "main branch with patches",
			version: Version{
				Major: 1,
				Minor: 0,
				Patch: 5,
			},
			expected: "1.0.5",
		},
		{
			name: "prerelease version",
			version: Version{
				Major:      1,
				Minor:      0,
				Patch:      2,
				Prerelease: "feature",
				Build:      0,
			},
			expected: "1.0.2-feature.0",
		},
		{
			name: "prerelease with multiple commits",
			version: Version{
				Major:      1,
				Minor:      0,
				Patch:      3,
				Prerelease: "bugfix",
				Build:      5,
			},
			expected: "1.0.3-bugfix.5",
		},
		{
			name: "prerelease with sanitized branch name",
			version: Version{
				Major:      1,
				Minor:      0,
				Patch:      1,
				Prerelease: "add-new-feature",
				Build:      2,
			},
			expected: "1.0.1-add-new-feature.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.version.String()
			if result != tt.expected {
				t.Errorf("Version.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}
