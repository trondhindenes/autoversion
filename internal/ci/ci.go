package ci

import (
	"fmt"
	"os"
	"strings"

	"github.com/trondhindenes/autoversion/internal/config"
)

// log writes a log message to stderr
func log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// WellKnownProviders contains default configurations for well-known CI providers
var WellKnownProviders = map[string]*config.CIProvider{
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

// DetectBranch attempts to detect the actual branch name from CI environment variables
// Returns the detected branch name and true if found, or empty string and false if not found
func DetectBranch(cfg *config.Config) (string, bool) {
	// If UseCIBranch is not enabled, return immediately
	if cfg.UseCIBranch == nil || !*cfg.UseCIBranch {
		return "", false
	}

	// Check if user has overridden the github-actions provider
	hasGitHubOverride := false
	if cfg.CIProviders != nil {
		if _, exists := cfg.CIProviders["github-actions"]; exists {
			hasGitHubOverride = true
		}
	}

	// Special handling for GitHub Actions (only if not overridden by user config)
	// GITHUB_HEAD_REF is set for pull requests
	// GITHUB_REF is set for all events (format: refs/heads/branch-name or refs/tags/tag-name)
	if !hasGitHubOverride {
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
	}

	// Merge user-configured providers with well-known providers
	providers := make(map[string]*config.CIProvider)

	// Start with well-known providers
	for k, v := range WellKnownProviders {
		providers[k] = v
	}

	// Override with user-configured providers
	if cfg.CIProviders != nil {
		for k, v := range cfg.CIProviders {
			providers[k] = v
		}
	}

	// Try each provider's environment variable
	for _, provider := range providers {
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
