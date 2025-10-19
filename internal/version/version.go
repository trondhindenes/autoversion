package version

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/trondhindenes/autoversion/internal/ci"
	"github.com/trondhindenes/autoversion/internal/config"
	"github.com/trondhindenes/autoversion/internal/defaults"
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

	// Check if this is a shallow clone
	log("Checking if repository is a shallow clone...")
	isShallow, err := repo.IsShallow()
	if err != nil {
		return "", fmt.Errorf("failed to check if repository is shallow: %w", err)
	}
	if isShallow {
		return "", fmt.Errorf("autoversion does not work with shallow clones. Please use 'git fetch --unshallow' to convert to a full clone, or clone without --depth")
	}
	log("Repository is not a shallow clone")

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

	// Determine main branches (with backward compatibility)
	mainBranches := cfg.MainBranches
	if len(mainBranches) == 0 {
		if cfg.MainBranch != "" {
			// Backward compatibility with old config
			mainBranches = []string{cfg.MainBranch}
		} else {
			mainBranches = defaults.MainBranches
		}
	}
	log("Configured main branches: %v", mainBranches)

	// Find which main branch exists in the repo
	mainBranch, err := repo.GetMainBranch(mainBranches)
	if err != nil {
		return "", fmt.Errorf("failed to find main branch: %w", err)
	}
	log("Using main branch: %s", mainBranch)

	// Get main branch behavior
	mainBranchBehavior := defaults.MainBranchBehavior
	if cfg.MainBranchBehavior != nil && *cfg.MainBranchBehavior != "" {
		mainBranchBehavior = *cfg.MainBranchBehavior
		log("Using configured main branch behavior: %s", mainBranchBehavior)
	} else {
		log("Using default main branch behavior: %s", mainBranchBehavior)
	}

	// Validate main branch behavior
	validBehavior := false
	for _, valid := range defaults.ValidMainBranchBehaviors {
		if mainBranchBehavior == valid {
			validBehavior = true
			break
		}
	}
	if !validBehavior {
		return "", fmt.Errorf("invalid mainBranchBehavior '%s': must be one of %v", mainBranchBehavior, defaults.ValidMainBranchBehaviors)
	}

	// Try to detect branch from CI environment first (for detached HEAD states in CI)
	var currentBranch string
	ciBranch, detected := ci.DetectBranch(cfg)
	if detected {
		log("CI branch detected: %s", ciBranch)
		currentBranch = ciBranch
	} else {
		// Fall back to git branch detection
		var err error
		currentBranch, err = repo.GetCurrentBranch()
		if err != nil {
			return "", fmt.Errorf("failed to get current branch: %w (note: this might be because you're in detached HEAD state - enable useCIBranch if in CI environment)", err)
		}
		log("Current git branch: %s", currentBranch)
	}

	// Check for most recent tag in history
	log("Looking for most recent tag in commit history...")
	mostRecentTag, commitsSinceTag, err := repo.GetMostRecentTag()
	if err != nil {
		return "", fmt.Errorf("failed to get most recent tag: %w", err)
	}

	// Determine the initial version to use when no tags exist
	initialVersionStr := defaults.InitialVersion
	if cfg.InitialVersion != nil && *cfg.InitialVersion != "" {
		initialVersionStr = *cfg.InitialVersion
		log("Using configured initial version: %s", initialVersionStr)
	} else {
		log("Using default initial version: %s", initialVersionStr)
	}

	// Parse and validate the initial version
	initialVersion, err := parseVersion(initialVersionStr)
	if err != nil {
		return "", fmt.Errorf("invalid initialVersion '%s': %w", initialVersionStr, err)
	}
	if !IsValidSemver(initialVersionStr) {
		return "", fmt.Errorf("initialVersion '%s' is not valid semver", initialVersionStr)
	}

	var baseVersion Version
	var useTagAsBase bool
	if mostRecentTag != "" {
		log("Found most recent tag in history: %s (%d commits ago)", mostRecentTag, commitsSinceTag)

		// Strip prefix and validate
		strippedTag := git.StripTagPrefix(mostRecentTag, tagPrefix)
		if tagPrefix != "" && strippedTag != mostRecentTag {
			log("Stripped tag prefix '%s': %s -> %s", tagPrefix, mostRecentTag, strippedTag)
		}

		if !IsValidSemver(strippedTag) {
			log("WARNING: Most recent tag '%s' is not valid semver (after stripping prefix), ignoring", strippedTag)
			log("Falling back to commit-count-based versioning with initial version %s", initialVersionStr)
			baseVersion = initialVersion
			useTagAsBase = false
			mostRecentTag = "" // Clear it so we use commit count
		} else {
			// Parse the version from the tag
			parsedVersion, err := parseVersion(strippedTag)
			if err != nil {
				log("WARNING: Failed to parse version from tag '%s': %v", strippedTag, err)
				log("Falling back to commit-count-based versioning with initial version %s", initialVersionStr)
				baseVersion = initialVersion
				useTagAsBase = false
				mostRecentTag = "" // Clear it so we use commit count
			} else {
				log("Using tag '%s' as base version", strippedTag)
				baseVersion = parsedVersion
				useTagAsBase = true
			}
		}
	} else {
		log("No tags found in commit history, using initial version %s", initialVersionStr)
		baseVersion = initialVersion
		useTagAsBase = false
	}

	// Get commit count on main branch
	mainCommitCount, err := repo.GetMainBranchCommitCount(mainBranch)
	if err != nil {
		return "", fmt.Errorf("failed to get commit count on main branch: %w", err)
	}
	log("Commit count on %s branch: %d", mainBranch, mainCommitCount)

	version := baseVersion

	isOnMainBranch := git.IsMainBranch(currentBranch, mainBranches)

	if isOnMainBranch {
		// On main branch
		log("On main branch, calculating version...")

		if mainBranchBehavior == "pre" {
			// In "pre" mode, non-tagged commits create prerelease versions
			log("Main branch behavior is 'pre': generating prerelease version")

			if useTagAsBase {
				// We have a tag in history
				// Determine the next version and create prerelease
				version.Patch = baseVersion.Patch + commitsSinceTag
				if commitsSinceTag > 0 {
					// There are commits since the tag, create prerelease
					version.Prerelease = defaults.PrereleaseID
					version.Build = commitsSinceTag - 1
					log("Created prerelease version %d commits since tag: %s", commitsSinceTag, version.String())
				} else {
					// We're exactly on the tag (should not reach here as tag check is earlier)
					log("On tag exactly, using tag version: %s", version.String())
				}
			} else {
				// No tags in history
				commitCount, err := repo.GetCommitCount()
				if err != nil {
					return "", fmt.Errorf("failed to get commit count: %w", err)
				}
				// First commit gets initial version as prerelease: 1.0.0-pre.0
				// Subsequent commits increment: 1.0.0-pre.1, 1.0.0-pre.2, etc.
				version.Prerelease = defaults.PrereleaseID
				version.Build = commitCount - 1
				log("Calculated prerelease version from commit count: %s", version.String())
			}
		} else {
			// In "release" mode (default), create release versions
			if useTagAsBase {
				// Increment patch version based on commits since the tag
				version.Patch += commitsSinceTag
				log("Incremented patch version by %d commits since tag: %s", commitsSinceTag, version.String())
			} else {
				// No valid tags in history, use commit count from start
				commitCount, err := repo.GetCommitCount()
				if err != nil {
					return "", fmt.Errorf("failed to get commit count: %w", err)
				}
				// Start from the initial version and increment by (commitCount - 1)
				// This way, first commit gets the initial version (e.g., 0.0.1), second gets 0.0.2, etc.
				if commitCount > 1 {
					version.Patch += (commitCount - 1)
				}
				log("Calculated version from commit count: %s", version.String())
			}
		}
	} else {
		// On feature branch: version is BASE.X-branchname.Y
		// X is the next patch version
		// Y is the number of commits on this branch since branching
		log("On feature branch '%s', calculating prerelease version...", currentBranch)

		// Use main branch commit count to determine next patch version
		if useTagAsBase {
			// The next version after the tag, considering main branch commits
			version.Patch = baseVersion.Patch + mainCommitCount
		} else {
			version.Patch = mainCommitCount
		}

		branchCommitCount, err := repo.GetCommitCountSinceBranchPoint(mainBranch, currentBranch)
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

// parseVersion parses a semver string into a Version struct
// Only parses MAJOR.MINOR.PATCH, ignores prerelease and build metadata
func parseVersion(semver string) (Version, error) {
	var v Version

	// Remove prerelease and build metadata for parsing
	parts := strings.Split(semver, "-")
	corePart := parts[0]

	parts = strings.Split(corePart, "+")
	corePart = parts[0]

	// Parse MAJOR.MINOR.PATCH
	parts = strings.Split(corePart, ".")
	if len(parts) != 3 {
		return v, fmt.Errorf("invalid semver format: %s", semver)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return v, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return v, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return v, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	v.Major = major
	v.Minor = minor
	v.Patch = patch

	return v, nil
}
