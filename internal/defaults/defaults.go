package defaults

import "github.com/trondhindenes/autoversion/internal/config"

// Application default values - single source of truth for all defaults
const (
	// Version-related defaults
	InitialVersion = "1.0.0" // Initial version when no tags exist in repository
	PrereleaseID   = "pre"   // Prerelease identifier for prerelease versions

	// Branch-related defaults
	MainBranchBehavior   = "release" // Default behavior for main branch: "release" or "pre"
	UnknownBranchName    = "unknown" // Fallback name for sanitized branches that become empty
	DefaultTagPrefix     = ""        // Default tag prefix (empty = no prefix)
	DefaultVersionPrefix = ""        // Default version prefix (empty = no prefix)
	DefaultUseCIBranch   = true      // Whether to detect branch from CI environment variables
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

// WellKnownCIProviders contains default configurations for well-known CI providers
// This is the source of truth for CI provider defaults
var WellKnownCIProviders = map[string]*config.CIProvider{
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
