package ci

import (
	"fmt"
	"os"
	"strings"

	"github.com/trondhindenes/autoversion/internal/config"
	"github.com/trondhindenes/autoversion/internal/defaults"
)

// log writes a log message to stderr
func log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// WellKnownProviders is deprecated - use defaults.WellKnownCIProviders instead
// This is kept for backward compatibility
var WellKnownProviders = defaults.WellKnownCIProviders

// DetectBranch attempts to detect the actual branch name from CI environment variables
// Returns the detected branch name and true if found, or empty string and false if not found
func DetectBranch(cfg *config.Config) (string, bool) {
	// If UseCIBranch is not enabled, return immediately
	if cfg.UseCIBranch == nil || !*cfg.UseCIBranch {
		return "", false
	}

	// Special handling for GitHub Actions
	// GITHUB_HEAD_REF is set for pull requests
	// GITHUB_REF is set for all events (format: refs/heads/branch-name or refs/tags/tag-name)
	if githubHeadRef := os.Getenv("GITHUB_HEAD_REF"); githubHeadRef != "" {
		log("Detected GitHub Actions (pull request)")
		log("Found branch name from GITHUB_HEAD_REF: %s", githubHeadRef)
		return githubHeadRef, true
	}
	if githubRef := os.Getenv("GITHUB_REF"); githubRef != "" {
		// Parse GITHUB_REF to extract branch name
		// Format: refs/heads/branch-name -> branch-name
		if strings.HasPrefix(githubRef, "refs/heads/") {
			branchName := strings.TrimPrefix(githubRef, "refs/heads/")
			log("Detected GitHub Actions (push)")
			log("Found branch name from GITHUB_REF: %s", branchName)
			return branchName, true
		}
		// If it's a tag, we still want to know
		if strings.HasPrefix(githubRef, "refs/tags/") {
			log("Detected GitHub Actions (tag event)")
			log("GITHUB_REF is a tag, not a branch: %s", githubRef)
		}
	}

	// Try each well-known provider's environment variable
	for _, provider := range defaults.WellKnownCIProviders {
		if provider.BranchEnvVar == "" {
			continue
		}

		branchName := os.Getenv(provider.BranchEnvVar)
		if branchName != "" {
			log("CI provider detected")
			log("Found branch name: %s", branchName)
			return branchName, true
		}
	}

	return "", false
}
