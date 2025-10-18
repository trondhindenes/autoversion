package config

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// CIProvider represents configuration for a specific CI provider
type CIProvider struct {
	BranchEnvVar string `json:"branchEnvVar" yaml:"branchEnvVar" jsonschema:"title=Branch Environment Variable,description=The environment variable that contains the actual branch name"`
}

// Config represents the application configuration
type Config struct {
	MainBranch    string                 `json:"mainBranch" yaml:"mainBranch" jsonschema:"title=Main Branch,description=The name of the main branch (default: main),default=main"`
	TagPrefix     *string                `json:"tagPrefix,omitempty" yaml:"tagPrefix,omitempty" jsonschema:"title=Tag Prefix,description=Prefix to strip from git tags (e.g. 'PRODUCT/' to convert 'PRODUCT/2.0.0' to '2.0.0'). Default is empty string"`
	VersionPrefix *string                `json:"versionPrefix,omitempty" yaml:"versionPrefix,omitempty" jsonschema:"title=Version Prefix,description=Prefix to add to the generated version output (e.g. 'v' to output 'v1.0.0' instead of '1.0.0'). Default is empty string"`
	UseCIBranch   *bool                  `json:"useCIBranch,omitempty" yaml:"useCIBranch,omitempty" jsonschema:"title=Use CI Branch,description=Whether to detect and use the actual branch name from CI environment variables. Useful for PR builds where CI checks out a temporary branch. Default is false"`
	CIProviders   map[string]*CIProvider `json:"ciProviders,omitempty" yaml:"ciProviders,omitempty" jsonschema:"title=CI Providers,description=Configuration for CI provider branch detection. Key is the provider name (e.g. 'github-actions')"`
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
