package version

import (
	"fmt"
	"os"

	"github.com/trondhindenes/autoversion/internal/ci"
	"github.com/trondhindenes/autoversion/internal/config"
	"github.com/trondhindenes/autoversion/internal/git"
)

// log writes a log message to stderr
func log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Version represents a semantic version
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      int
}

// String returns the string representation of the version
func (v Version) String() string {
	if v.Prerelease != "" {
		return fmt.Sprintf("%d.%d.%d-%s.%d", v.Major, v.Minor, v.Patch, v.Prerelease, v.Build)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Calculate calculates the version based on the current git state
func Calculate(mainBranch, tagPrefix string) (string, error) {
	cfg := &config.Config{
		MainBranch: mainBranch,
		TagPrefix:  &tagPrefix,
	}
	return CalculateWithConfig(cfg)
}

// CalculateWithConfig calculates the version based on the current git state and configuration
func CalculateWithConfig(cfg *config.Config) (string, error) {
	log("Opening git repository...")
	repo, err := git.OpenRepo(".")
	if err != nil {
		return "", fmt.Errorf("failed to open git repository: %w", err)
	}

	// Check for tags first - tags take precedence over everything
	log("Checking for git tags on current commit...")
	tag, err := repo.GetTagOnCurrentCommit()
	if err != nil {
		return "", fmt.Errorf("failed to get tag on current commit: %w", err)
	}

	tagPrefix := ""
	if cfg.TagPrefix != nil {
		tagPrefix = *cfg.TagPrefix
	}

	if tag != "" {
		log("Found git tag: %s", tag)

		// Strip the configured prefix
		version := git.StripTagPrefix(tag, tagPrefix)
		if tagPrefix != "" && version != tag {
			log("Stripped tag prefix '%s': %s -> %s", tagPrefix, tag, version)
		}

		// Validate that the stripped tag is valid semver
		if !IsValidSemver(version) {
			log("WARNING: Tag '%s' is not valid semver (after stripping prefix), ignoring tag", version)
			log("Falling back to calculated version based on commit count")
			// Continue with normal version calculation
		} else {
			log("Using tag as version: %s", version)
			result := applyVersionPrefix(version, cfg)
			if result != version {
				log("Applied version prefix: %s -> %s", version, result)
			}
			return result, nil
		}
	} else {
		log("No git tag found on current commit")
	}

	// No tag found, calculate version based on branch and commit count
	log("Calculating version based on commit count...")
	mainBranch := cfg.MainBranch
	if mainBranch == "" {
		mainBranch = "main"
		log("Using default main branch: %s", mainBranch)
	} else {
		log("Using configured main branch: %s", mainBranch)
	}

	// Try to detect branch from CI environment
	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	log("Current git branch: %s", currentBranch)

	// Check if we should use CI branch detection
	ciBranch, detected := ci.DetectBranch(cfg)
	if detected {
		log("CI branch detected: %s (overriding git branch %s)", ciBranch, currentBranch)
		currentBranch = ciBranch
	}

	// Get commit count on main branch
	mainCommitCount, err := repo.GetMainBranchCommitCount(mainBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get commit count on main branch: %w", err)
	}
	log("Commit count on %s branch: %d", mainBranch, mainCommitCount)

	version := Version{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}

	if currentBranch == mainBranch {
		// On main branch: version is 1.0.0, 1.0.1, 1.0.2, etc.
		log("On main branch, calculating version...")
		commitCount, err := repo.GetCommitCount()
		if err != nil {
			return "", fmt.Errorf("failed to get commit count: %w", err)
		}

		if commitCount > 0 {
			version.Patch = commitCount - 1
		}
		log("Calculated version for main branch: %s", version.String())
	} else {
		// On feature branch: version is 1.0.X-branchname.Y
		// X is the next patch version (mainCommitCount)
		// Y is the number of commits on this branch since branching
		log("On feature branch '%s', calculating prerelease version...", currentBranch)
		version.Patch = mainCommitCount

		branchCommitCount, err := repo.GetCommitCountSinceBranchPoint(mainBranch)
		if err != nil {
			return "", fmt.Errorf("failed to get commit count since branch point: %w", err)
		}

		sanitizedBranch := git.SanitizeBranchName(currentBranch)
		if sanitizedBranch != currentBranch {
			log("Sanitized branch name: %s -> %s", currentBranch, sanitizedBranch)
		}
		version.Prerelease = sanitizedBranch
		version.Build = branchCommitCount
		log("Commits on feature branch since branching: %d", branchCommitCount)
		log("Calculated prerelease version: %s", version.String())
	}

	result := applyVersionPrefix(version.String(), cfg)
	if result != version.String() {
		log("Applied version prefix: %s -> %s", version.String(), result)
	}
	log("Final version: %s", result)
	return result, nil
}

// applyVersionPrefix adds the configured version prefix to the version string
func applyVersionPrefix(version string, cfg *config.Config) string {
	if cfg.VersionPrefix != nil && *cfg.VersionPrefix != "" {
		return *cfg.VersionPrefix + version
	}
	return version
}
