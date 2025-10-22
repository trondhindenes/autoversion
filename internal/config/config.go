package config

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// Config represents the application configuration
type Config struct {
	MainBranch            string   `json:"mainBranch,omitempty" yaml:"mainBranch,omitempty" jsonschema:"title=Main Branch (deprecated),description=Deprecated: Use mainBranches instead. The name of the main branch"`
	MainBranches          []string `json:"mainBranches,omitempty" yaml:"mainBranches,omitempty" jsonschema:"title=Main Branches,description=List of branch names to treat as main branches (default: ['main' 'master']). The first matching branch found is used"`
	MainBranchBehavior    *string  `json:"mainBranchBehavior,omitempty" yaml:"mainBranchBehavior,omitempty" jsonschema:"title=Main Branch Behavior,description=Behavior for non-tagged commits on main branch: 'release' (default) creates release versions '1.0.0' or 'pre' creates prerelease versions '1.0.0-pre.0',enum=release,enum=pre"`
	TagPrefix             *string  `json:"tagPrefix,omitempty" yaml:"tagPrefix,omitempty" jsonschema:"title=Tag Prefix,description=Prefix to strip from git tags (e.g. 'PRODUCT/' to convert 'PRODUCT/2.0.0' to '2.0.0'). Default is empty string"`
	VersionPrefix         *string  `json:"versionPrefix,omitempty" yaml:"versionPrefix,omitempty" jsonschema:"title=Version Prefix,description=Prefix to add to the generated version output (e.g. 'v' to output 'v1.0.0' instead of '1.0.0'). Default is empty string"`
	InitialVersion        *string  `json:"initialVersion,omitempty" yaml:"initialVersion,omitempty" jsonschema:"title=Initial Version,description=The initial version to use when no tags exist in the repository (e.g. '0.0.1' or '1.0.0'). Default is '1.0.0'. Must be valid semver"`
	UseCIBranch           *bool    `json:"useCIBranch,omitempty" yaml:"useCIBranch,omitempty" jsonschema:"title=Use CI Branch,description=Whether to detect and use the actual branch name from CI environment variables. Useful for PR builds where CI checks out a temporary branch. Default is false"`
	FailOnOutdatedBase    *bool    `json:"failOnOutdatedBase,omitempty" yaml:"failOnOutdatedBase,omitempty" jsonschema:"title=Fail On Outdated Base,description=When running on a feature branch if true and the main branch has been tagged after this branch diverged autoversion will exit with an error instead of just warning. Default is false"`
	OutdatedBaseCheckMode *string  `json:"outdatedBaseCheckMode,omitempty" yaml:"outdatedBaseCheckMode,omitempty" jsonschema:"title=Outdated Base Check Mode,description=Controls what triggers the outdated base warning/error on feature branches: 'tagged' (default) only warns when main has new tags or 'all' warns when main has any new commits since branching,enum=tagged,enum=all"`
}

// GenerateSchema generates a JSON schema for the configuration
func GenerateSchema() (string, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	schema := reflector.Reflect(&Config{})
	schema.Title = "Autoversion Configuration"
	schema.Description = "Configuration file for autoversion tool"

	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}
	return string(schemaBytes), nil
}
