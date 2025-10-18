package version

import "regexp"

// semverRegex matches semantic versions according to semver 2.0.0
// Format: MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]
var semverRegex = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// IsValidSemver checks if a string is a valid semantic version
func IsValidSemver(version string) bool {
	return semverRegex.MatchString(version)
}
