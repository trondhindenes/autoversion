package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/trondhindenes/autoversion/internal/config"
	"github.com/trondhindenes/autoversion/internal/defaults"
	"github.com/trondhindenes/autoversion/internal/ghactions"
	"github.com/trondhindenes/autoversion/internal/version"
)

var (
	// Version is set at build time via -ldflags
	Version    = "0.0.1-dev"
	cfgFile    string
	configFlag []string

	// gh-versions command flags
	ghWorkflow  string
	ghJob       string
	ghStep      string
	ghLimit     int
	ghOutputFmt string
	ghVerbose   bool

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
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}
	ghVersionsCmd = &cobra.Command{
		Use:   "gh-versions",
		Short: "Get calculated versions from GitHub Actions workflow runs",
		Long: `Fetches version information from recent GitHub Actions workflow runs.

This command uses the gh CLI to list recent workflow runs and extract the
calculated version from the "Final version:" log output.

Requires:
  - gh CLI installed and authenticated
  - A workflow that outputs version info in the format:
    Final version: {"semver":"1.0.0",...}

Examples:
  # Get versions from all workflows
  autoversion gh-versions

  # Get versions from a specific workflow
  autoversion gh-versions -w "CI"

  # Get versions from a specific workflow and job
  autoversion gh-versions -w "CI" -j "build"

  # Limit results
  autoversion gh-versions -w "CI" -L 5`,
		Run: runGhVersions,
	}
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .autoversion.yaml)")
	rootCmd.PersistentFlags().StringArrayVar(&configFlag, "config-flag", []string{}, "override config setting (format: key=value, can be used multiple times)")
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(ghVersionsCmd)

	// gh-versions command flags
	ghVersionsCmd.Flags().StringVarP(&ghWorkflow, "workflow", "w", "", "workflow name or filename (e.g., 'CI' or 'ci.yml')")
	ghVersionsCmd.Flags().StringVarP(&ghJob, "job", "j", "", "job name to filter logs (e.g., 'build')")
	ghVersionsCmd.Flags().StringVarP(&ghStep, "step", "s", "calculate version", "step name containing version output")
	ghVersionsCmd.Flags().IntVarP(&ghLimit, "limit", "L", 5, "maximum number of runs to fetch")
	ghVersionsCmd.Flags().StringVarP(&ghOutputFmt, "output", "o", "table", "output format: table, json")
	ghVersionsCmd.Flags().BoolVarP(&ghVerbose, "verbose", "v", false, "print verbose progress information")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// List all files in current directory and find config file
		entries, err := os.ReadDir(".")
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				filename := entry.Name()
				lowerFilename := strings.ToLower(filename)
				if lowerFilename == ".autoversion.yaml" || lowerFilename == ".autoversion.yml" {
					viper.SetConfigFile(filename)
					break
				}
			}
		}

		// If no config file found, fallback to viper's default behavior
		if viper.ConfigFileUsed() == "" {
			viper.AddConfigPath(".")
			viper.SetConfigName(".autoversion")
			viper.SetConfigType("yaml")
		}
	}

	viper.SetDefault("mainBranches", defaults.MainBranches)
	viper.SetDefault("mainBranchBehavior", defaults.MainBranchBehavior)
	viper.SetDefault("mode", defaults.DefaultMode)
	viper.SetDefault("tagPrefix", defaults.DefaultTagPrefix)
	viper.SetDefault("versionPrefix", defaults.DefaultVersionPrefix)
	viper.SetDefault("initialVersion", defaults.InitialVersion)
	viper.SetDefault("useCIBranch", defaults.DefaultUseCIBranch)
	viper.SetDefault("failOnOutdatedBase", defaults.DefaultFailOnOutdated)
	viper.SetDefault("outdatedBaseCheckMode", defaults.DefaultOutdatedCheckMode)

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Process command-line config overrides
	for _, override := range configFlag {
		parts := strings.SplitN(override, "=", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Warning: ignoring invalid config-flag format '%s' (expected key=value)\n", override)
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Try to parse as boolean
		if boolVal, err := strconv.ParseBool(value); err == nil {
			viper.Set(key, boolVal)
			continue
		}

		// Try to parse as int
		if intVal, err := strconv.Atoi(value); err == nil {
			viper.Set(key, intVal)
			continue
		}

		// Treat as string
		viper.Set(key, value)
	}
}

func run(cmd *cobra.Command, args []string) {
	// Build config from viper settings
	cfg := &config.Config{}

	// Handle mainBranches (with backward compatibility for mainBranch)
	if viper.IsSet("mainBranch") {
		// Backward compatibility: if old mainBranch is set, use it
		cfg.MainBranch = viper.GetString("mainBranch")
		cfg.MainBranches = []string{cfg.MainBranch}
	} else if viper.IsSet("mainBranches") {
		cfg.MainBranches = viper.GetStringSlice("mainBranches")
	} else {
		cfg.MainBranches = defaults.MainBranches
	}

	// Handle optional fields
	if viper.IsSet("mainBranchBehavior") {
		behavior := viper.GetString("mainBranchBehavior")
		cfg.MainBranchBehavior = &behavior
	}

	if viper.IsSet("mode") {
		mode := viper.GetString("mode")
		cfg.Mode = &mode
	}

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

	if viper.IsSet("failOnOutdatedBase") {
		failOnOutdatedBase := viper.GetBool("failOnOutdatedBase")
		cfg.FailOnOutdatedBase = &failOnOutdatedBase
	}

	if viper.IsSet("outdatedBaseCheckMode") {
		outdatedBaseCheckMode := viper.GetString("outdatedBaseCheckMode")
		cfg.OutdatedBaseCheckMode = &outdatedBaseCheckMode
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

func runGhVersions(cmd *cobra.Command, args []string) {
	versions, err := ghactions.GetVersionsFromRuns(ghWorkflow, ghJob, ghStep, ghLimit, ghVerbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(versions) == 0 {
		fmt.Fprintln(os.Stderr, "No versions found in workflow runs")
		os.Exit(0)
	}

	switch ghOutputFmt {
	case "json":
		output, err := json.MarshalIndent(versions, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(output))
	default:
		// Table format
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "BRANCH\tCOMMIT\tVERSION\tWORKFLOW\tJOB\tSTATUS\tRUN")
		for _, v := range versions {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t#%d\n", v.Branch, v.CommitSHA, v.Version, v.Workflow, v.Job, v.Conclusion, v.RunNumber)
		}
		w.Flush()
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
