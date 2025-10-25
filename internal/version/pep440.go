package version

import (
	"fmt"
	"regexp"
	"strings"
)

// ConvertToPEP440 converts a semver version string to PEP 440 format
// Examples:
//   - "1.0.2-setup-build.1" -> "1.0.2a1"
//   - "1.0.0-pre.5" -> "1.0.0a5"
//   - "2.3.4-feature-auth.10" -> "2.3.4a10"
//   - "1.0.0" -> "1.0.0" (no change for release versions)
func ConvertToPEP440(semver string) (string, error) {
	// Release versions (no prerelease) remain unchanged
	if !strings.Contains(semver, "-") {
		return semver, nil
	}

	// Parse the semver: MAJOR.MINOR.PATCH-PRERELEASE.BUILD
	// The prerelease part can contain hyphens (e.g., "setup-build.1" or "feature-auth.10")
	dashIndex := strings.Index(semver, "-")
	if dashIndex == -1 {
		return semver, nil
	}

	corePart := semver[:dashIndex]         // e.g., "1.0.2"
	prereleasePart := semver[dashIndex+1:] // e.g., "setup-build.1" or "feature-auth.10"

	// Extract the build number from prerelease (everything after the last dot)
	// e.g., "setup-build.1" -> build = "1", identifier = "setup-build"
	lastDotIndex := strings.LastIndex(prereleasePart, ".")
	if lastDotIndex == -1 {
		return "", fmt.Errorf("invalid prerelease format (missing build number): %s", semver)
	}

	buildNumber := prereleasePart[lastDotIndex+1:]
	// PEP 440 uses 'a' for alpha versions (similar to prerelease)
	// Format: MAJOR.MINOR.PATCHaN where N is the build number
	pep440Version := fmt.Sprintf("%sa%s", corePart, buildNumber)

	return pep440Version, nil
}

// IsValidPEP440 checks if a string is a valid PEP 440 version
// This is a simplified check for the versions we generate
// Full PEP 440 spec: https://peps.python.org/pep-0440/
func IsValidPEP440(version string) bool {
	// Simple regex for the versions we generate: X.Y.Z or X.Y.ZaN
	pep440Regex := regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(a\d+)?$`)
	return pep440Regex.MatchString(version)
}
