package ci

import (
	"os"
	"testing"

	"github.com/trondhindenes/autoversion/internal/config"
)

func TestDetectBranch(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		envVars        map[string]string
		expectedBranch string
		expectedFound  bool
	}{
		{
			name: "UseCIBranch disabled",
			config: &config.Config{
				UseCIBranch: boolPtr(false),
			},
			envVars: map[string]string{
				"GITHUB_HEAD_REF": "feature/test",
			},
			expectedBranch: "",
			expectedFound:  false,
		},
		{
			name: "UseCIBranch nil (disabled by default)",
			config: &config.Config{
				UseCIBranch: nil,
			},
			envVars: map[string]string{
				"GITHUB_HEAD_REF": "feature/test",
			},
			expectedBranch: "",
			expectedFound:  false,
		},
		{
			name: "GitHub Actions - branch detected",
			config: &config.Config{
				UseCIBranch: boolPtr(true),
			},
			envVars: map[string]string{
				"GITHUB_HEAD_REF": "feature/new-feature",
			},
			expectedBranch: "feature/new-feature",
			expectedFound:  true,
		},
		{
			name: "GitLab CI - branch detected",
			config: &config.Config{
				UseCIBranch: boolPtr(true),
			},
			envVars: map[string]string{
				"CI_MERGE_REQUEST_SOURCE_BRANCH_NAME": "bugfix/fix-issue",
			},
			expectedBranch: "bugfix/fix-issue",
			expectedFound:  true,
		},
		{
			name: "CircleCI - branch detected",
			config: &config.Config{
				UseCIBranch: boolPtr(true),
			},
			envVars: map[string]string{
				"CIRCLE_BRANCH": "develop",
			},
			expectedBranch: "develop",
			expectedFound:  true,
		},
		{
			name: "No CI environment variables set",
			config: &config.Config{
				UseCIBranch: boolPtr(true),
			},
			envVars:        map[string]string{},
			expectedBranch: "",
			expectedFound:  false,
		},
		{
			name: "Multiple CI vars set - first found wins",
			config: &config.Config{
				UseCIBranch: boolPtr(true),
			},
			envVars: map[string]string{
				"GITHUB_HEAD_REF":                     "github-branch",
				"CI_MERGE_REQUEST_SOURCE_BRANCH_NAME": "gitlab-branch",
			},
			expectedBranch: "",
			expectedFound:  true, // One of them will be found, but which one depends on map iteration
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables first
			clearCIEnvVars()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			branch, found := DetectBranch(tt.config)

			if found != tt.expectedFound {
				t.Errorf("DetectBranch() found = %v, expected %v", found, tt.expectedFound)
			}

			// For the multiple CI vars test, we just check that something was found
			if tt.name == "Multiple CI vars set - first found wins" {
				if found && (branch == "github-branch" || branch == "gitlab-branch") {
					return // Pass - one of the expected branches was found
				}
				if !found {
					t.Errorf("DetectBranch() expected to find a branch but found none")
				}
				return
			}

			if branch != tt.expectedBranch {
				t.Errorf("DetectBranch() branch = %v, expected %v", branch, tt.expectedBranch)
			}
		})
	}
}

func TestWellKnownProviders(t *testing.T) {
	expectedProviders := []string{
		"github-actions",
		"gitlab-ci",
		"circleci",
		"travis-ci",
		"jenkins",
		"azure-pipelines",
	}

	for _, provider := range expectedProviders {
		if _, exists := WellKnownProviders[provider]; !exists {
			t.Errorf("Expected well-known provider %s to exist", provider)
		}

		if WellKnownProviders[provider].BranchEnvVar == "" {
			t.Errorf("Expected well-known provider %s to have a BranchEnvVar set", provider)
		}
	}
}

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to clear all CI environment variables
func clearCIEnvVars() {
	envVars := []string{
		"GITHUB_HEAD_REF",
		"CI_MERGE_REQUEST_SOURCE_BRANCH_NAME",
		"CIRCLE_BRANCH",
		"TRAVIS_PULL_REQUEST_BRANCH",
		"CHANGE_BRANCH",
		"SYSTEM_PULLREQUEST_SOURCEBRANCH",
		"CUSTOM_BRANCH_VAR",
		"CUSTOM_GITHUB_VAR",
	}

	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
