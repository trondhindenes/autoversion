package version

import (
	"testing"
)

func TestConvertToPEP440(t *testing.T) {
	tests := []struct {
		name        string
		semver      string
		expected    string
		shouldError bool
	}{
		{
			name:        "prerelease with build number",
			semver:      "1.0.2-setup-build.1",
			expected:    "1.0.2a1",
			shouldError: false,
		},
		{
			name:        "pre with build number",
			semver:      "1.0.0-pre.5",
			expected:    "1.0.0a5",
			shouldError: false,
		},
		{
			name:        "feature branch prerelease",
			semver:      "2.3.4-feature-auth.10",
			expected:    "2.3.4a10",
			shouldError: false,
		},
		{
			name:        "release version unchanged",
			semver:      "1.0.0",
			expected:    "1.0.0",
			shouldError: false,
		},
		{
			name:        "release version with higher numbers unchanged",
			semver:      "3.2.1",
			expected:    "3.2.1",
			shouldError: false,
		},
		{
			name:        "complex prerelease identifier",
			semver:      "1.2.3-feature-user-auth-system.42",
			expected:    "1.2.3a42",
			shouldError: false,
		},
		{
			name:        "zero build number",
			semver:      "1.0.0-pre.0",
			expected:    "1.0.0a0",
			shouldError: false,
		},
		{
			name:        "invalid - missing build number",
			semver:      "1.0.0-pre",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "invalid - multiple dashes",
			semver:      "1.0.0-pre-test-more",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertToPEP440(tt.semver)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.semver)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.semver, err)
				}
				if result != tt.expected {
					t.Errorf("ConvertToPEP440(%q) = %q, want %q", tt.semver, result, tt.expected)
				}
			}
		})
	}
}

func TestIsValidPEP440(t *testing.T) {
	tests := []struct {
		name    string
		version string
		valid   bool
	}{
		{
			name:    "valid release version",
			version: "1.0.0",
			valid:   true,
		},
		{
			name:    "valid alpha version",
			version: "1.0.0a1",
			valid:   true,
		},
		{
			name:    "valid alpha with larger numbers",
			version: "10.20.30a99",
			valid:   true,
		},
		{
			name:    "valid release with zeros",
			version: "0.0.0",
			valid:   true,
		},
		{
			name:    "invalid - semver prerelease format",
			version: "1.0.0-pre.1",
			valid:   false,
		},
		{
			name:    "invalid - missing patch",
			version: "1.0",
			valid:   false,
		},
		{
			name:    "invalid - beta instead of alpha",
			version: "1.0.0b1",
			valid:   false,
		},
		{
			name:    "invalid - leading zeros",
			version: "01.02.03",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPEP440(tt.version)
			if result != tt.valid {
				t.Errorf("IsValidPEP440(%q) = %v, want %v", tt.version, result, tt.valid)
			}
		})
	}
}
