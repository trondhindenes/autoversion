package ci

import (
	"os"

	"github.com/trondhindenes/autoversion/internal/config"
)

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
			return branchName, true
		}
	}

	return "", false
}
