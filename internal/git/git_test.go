package git

import (
	"testing"
)

func TestStripTagPrefix(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		prefix   string
		expected string
	}{
		{
			name:     "no prefix configured",
			tag:      "2.0.0",
			prefix:   "",
			expected: "2.0.0",
		},
		{
			name:     "prefix matches",
			tag:      "PRODUCT/2.0.0",
			prefix:   "PRODUCT/",
			expected: "2.0.0",
		},
		{
			name:     "prefix does not match",
			tag:      "v2.0.0",
			prefix:   "PRODUCT/",
			expected: "v2.0.0",
		},
		{
			name:     "v prefix",
			tag:      "v1.5.3",
			prefix:   "v",
			expected: "1.5.3",
		},
		{
			name:     "complex prefix",
			tag:      "myapp/release/3.2.1",
			prefix:   "myapp/release/",
			expected: "3.2.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripTagPrefix(tt.tag, tt.prefix)
			if result != tt.expected {
				t.Errorf("StripTagPrefix(%q, %q) = %q, want %q", tt.tag, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "feature branch with prefix",
			input:    "feature/add-new-feature",
			expected: "add-new-feature",
		},
		{
			name:     "bugfix branch with prefix",
			input:    "bugfix/fix-crash",
			expected: "fix-crash",
		},
		{
			name:     "branch with slashes",
			input:    "feature/user/login",
			expected: "user-login",
		},
		{
			name:     "branch with underscores",
			input:    "fix_memory_leak",
			expected: "fix-memory-leak",
		},
		{
			name:     "branch with special characters",
			input:    "feature/add@new#feature",
			expected: "add-new-feature",
		},
		{
			name:     "branch with multiple hyphens",
			input:    "feature---test",
			expected: "feature-test",
		},
		{
			name:     "simple branch name",
			input:    "develop",
			expected: "develop",
		},
		{
			name:     "uppercase branch name",
			input:    "FEATURE/TEST",
			expected: "feature-test",
		},
		{
			name:     "empty after sanitization",
			input:    "///",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
