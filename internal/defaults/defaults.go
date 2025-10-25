package defaults

// Application default values - single source of truth for all defaults
const (
	// Version-related defaults
	InitialVersion = "1.0.0"  // Initial version when no tags exist in repository
	PrereleaseID   = "pre"    // Prerelease identifier for prerelease versions
	DefaultMode    = "semver" // Default version format mode: "semver" or "pep440"
	ModeSemver     = "semver" // Semver mode constant
	ModePep440     = "pep440" // PEP 440 mode constant

	// Branch-related defaults
	MainBranchBehavior       = "release" // Default behavior for main branch: "release" or "pre"
	UnknownBranchName        = "unknown" // Fallback name for sanitized branches that become empty
	DefaultTagPrefix         = ""        // Default tag prefix (empty = no prefix)
	DefaultVersionPrefix     = ""        // Default version prefix (empty = no prefix)
	DefaultUseCIBranch       = true      // Whether to detect branch from CI environment variables
	DefaultFailOnOutdated    = false     // Whether to fail (vs warn) when feature branch base is outdated
	DefaultOutdatedCheckMode = "tagged"  // Default mode for outdated base check: "tagged" or "all"
	OutdatedCheckModeTagged  = "tagged"  // Check mode: only warn on new tags
	OutdatedCheckModeAll     = "all"     // Check mode: warn on any new commits
)

// Branch name prefixes that are automatically stripped during sanitization
var BranchPrefixesToStrip = []string{
	"feature/",
	"bugfix/",
	"hotfix/",
	"release/",
}

// Main branch names to check (in order of preference)
var MainBranches = []string{"main", "master"}

// ValidMainBranchBehaviors are the allowed values for main branch behavior
var ValidMainBranchBehaviors = []string{"release", "pre"}

// ValidModes are the allowed values for version mode
var ValidModes = []string{ModeSemver, ModePep440}

// ValidOutdatedCheckModes are the allowed values for outdated base check mode
var ValidOutdatedCheckModes = []string{OutdatedCheckModeTagged, OutdatedCheckModeAll}

// CIProvider represents configuration for a specific CI provider
type CIProvider struct {
	BranchEnvVar string
}

// WellKnownCIProviders contains default configurations for well-known CI providers
// This is the source of truth for CI provider defaults
var WellKnownCIProviders = map[string]*CIProvider{
	"github-actions": {
		BranchEnvVar: "GITHUB_HEAD_REF",
	},
	"gitlab-ci": {
		BranchEnvVar: "CI_MERGE_REQUEST_SOURCE_BRANCH_NAME",
	},
	"circleci": {
		BranchEnvVar: "CIRCLE_BRANCH",
	},
	"travis-ci": {
		BranchEnvVar: "TRAVIS_PULL_REQUEST_BRANCH",
	},
	"jenkins": {
		BranchEnvVar: "CHANGE_BRANCH",
	},
	"azure-pipelines": {
		BranchEnvVar: "SYSTEM_PULLREQUEST_SOURCEBRANCH",
	},
}
