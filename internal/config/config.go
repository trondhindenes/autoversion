package config

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// Config represents the application configuration
type Config struct {
	MainBranch string  `json:"mainBranch" yaml:"mainBranch" jsonschema:"title=Main Branch,description=The name of the main branch (default: main),default=main"`
	TagPrefix  *string `json:"tagPrefix,omitempty" yaml:"tagPrefix,omitempty" jsonschema:"title=Tag Prefix,description=Prefix to strip from git tags (e.g. 'PRODUCT/' to convert 'PRODUCT/2.0.0' to '2.0.0'). Default is empty string"`
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
