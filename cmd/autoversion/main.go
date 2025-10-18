package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/trondhindenes/autoversion/internal/config"
	"github.com/trondhindenes/autoversion/internal/version"
)

var (
	cfgFile string
	rootCmd = &cobra.Command{
		Use:   "autoversion",
		Short: "Automatically generate semantic versions based on git repository state",
		Long: `autoversion is a CLI tool that generates semantic versions based on the state of a git repository.
It calculates versions for the main branch (e.g., 1.0.0, 1.0.1) and prerelease versions for feature branches (e.g., 1.0.2-feature.0).`,
		Run: run,
	}
	schemaCmd = &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON schema for the configuration file",
		Run:   runSchema,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .autoversion.yaml)")
	rootCmd.AddCommand(schemaCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigName(".autoversion")
		viper.SetConfigType("yaml")
	}

	viper.SetDefault("mainBranch", "main")
	viper.SetDefault("tagPrefix", "")
	viper.SetDefault("versionPrefix", "")
	viper.SetDefault("initialVersion", "1.0.0")
	viper.SetDefault("useCIBranch", false)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func run(cmd *cobra.Command, args []string) {
	// Build config from viper settings
	cfg := &config.Config{
		MainBranch: viper.GetString("mainBranch"),
	}

	// Handle optional fields
	if viper.IsSet("tagPrefix") {
		tagPrefix := viper.GetString("tagPrefix")
		cfg.TagPrefix = &tagPrefix
	}

	if viper.IsSet("versionPrefix") {
		versionPrefix := viper.GetString("versionPrefix")
		cfg.VersionPrefix = &versionPrefix
	}

	if viper.IsSet("initialVersion") {
		initialVersion := viper.GetString("initialVersion")
		cfg.InitialVersion = &initialVersion
	}

	if viper.IsSet("useCIBranch") {
		useCIBranch := viper.GetBool("useCIBranch")
		cfg.UseCIBranch = &useCIBranch
	}

	if viper.IsSet("ciProviders") {
		ciProviders := viper.GetStringMap("ciProviders")
		cfg.CIProviders = make(map[string]*config.CIProvider)
		for name, providerData := range ciProviders {
			if providerMap, ok := providerData.(map[string]interface{}); ok {
				if branchEnvVar, ok := providerMap["branchEnvVar"].(string); ok {
					cfg.CIProviders[name] = &config.CIProvider{
						BranchEnvVar: branchEnvVar,
					}
				}
			}
		}
	}

	ver, err := version.CalculateWithConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(ver)
}

func runSchema(cmd *cobra.Command, args []string) {
	schema, err := config.GenerateSchema()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(schema)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
