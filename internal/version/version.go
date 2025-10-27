package version

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/trondhindenes/autoversion/internal/ci"
	"github.com/trondhindenes/autoversion/internal/config"
	"github.com/trondhindenes/autoversion/internal/defaults"
	"github.com/trondhindenes/autoversion/internal/git"
)

// VersionOutput represents the JSON output structure for version information
type VersionOutput struct {
	Semver           string `json:"semver"`
	SemverWithPrefix string `json:"semverWithPrefix"`
	Pep440           string `json:"pep440"`
	Pep440WithPrefix string `json:"pep440WithPrefix"`
	Major            int    `json:"major"`
	Minor            int    `json:"minor"`
	Patch            int    `json:"patch"`
	IsRelease        bool   `json:"isRelease"`
}

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
			// Apply mode conversion (which handles prefix internally for JSON mode)
			modeVersion, err := applyVersionMode(version, cfg)
			if err != nil {
				return "", fmt.Errorf("failed to apply version mode: %w", err)
			}
			// For non-JSON modes, apply prefix here
			mode := defaults.DefaultMode
			if cfg.Mode != nil && *cfg.Mode != "" {
				mode = *cfg.Mode
			}
			if mode != defaults.ModeJson {
				result := applyVersionPrefix(modeVersion, cfg)
				if result != modeVersion {
					log("Applied version prefix: %s -> %s", modeVersion, result)
				}
				return result, nil
			}
			return modeVersion, nil
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
	mostRecentTag, commitsSinceTag, err := repo.GetMostRecentTag(tagPrefix)
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
	var tagNotInBranchHistory bool
	if mostRecentTag != "" {
		// GetMostRecentTag now only returns tags in the current branch's history
		log("Found most recent tag in history: %s (%d commits ago)", mostRecentTag, commitsSinceTag)
		tagNotInBranchHistory = false

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
		tagNotInBranchHistory = false
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
		// X is the next patch version (base + 1 + commits on main since branching)
		// Y is the number of commits on this branch since branching
		log("On feature branch '%s', calculating prerelease version...", currentBranch)

		// Calculate how many commits have been added to main since this branch diverged
		mainCommitsSinceBranch, err := repo.GetMainBranchCommitsSinceBranchPoint(mainBranch, currentBranch)
		if err != nil {
			return "", fmt.Errorf("failed to get main branch commits since branch point: %w", err)
		}
		log("Commits on main branch since branching: %d", mainCommitsSinceBranch)

		// Determine the outdated check mode
		outdatedCheckMode := defaults.DefaultOutdatedCheckMode
		if cfg.OutdatedBaseCheckMode != nil && *cfg.OutdatedBaseCheckMode != "" {
			outdatedCheckMode = *cfg.OutdatedBaseCheckMode
			log("Using configured outdated base check mode: %s", outdatedCheckMode)
		} else {
			log("Using default outdated base check mode: %s", outdatedCheckMode)
		}

		// Validate outdated check mode
		validCheckMode := false
		for _, valid := range defaults.ValidOutdatedCheckModes {
			if outdatedCheckMode == valid {
				validCheckMode = true
				break
			}
		}
		if !validCheckMode {
			return "", fmt.Errorf("invalid outdatedBaseCheckMode '%s': must be one of %v", outdatedCheckMode, defaults.ValidOutdatedCheckModes)
		}

		// Check for outdated base based on the configured mode
		var isOutdated bool
		var outdatedReason string

		if outdatedCheckMode == defaults.OutdatedCheckModeTagged {
			// Check only for new tags
			hasNewTags, newTag, err := repo.CheckMainBranchHasNewTagsSinceBranchPoint(mainBranch, currentBranch)
			if err != nil {
				// Don't fail on this check, just log the error
				log("Warning: failed to check for new tags on main branch: %v", err)
			} else if hasNewTags {
				isOutdated = true
				outdatedReason = fmt.Sprintf("new tag(s) since this branch diverged (most recent: %s)", newTag)
			}
		} else if outdatedCheckMode == defaults.OutdatedCheckModeAll {
			// Check for any new commits
			hasNewCommits, err := repo.CheckMainBranchHasNewCommitsSinceBranchPoint(mainBranch, currentBranch)
			if err != nil {
				// Don't fail on this check, just log the error
				log("Warning: failed to check for new commits on main branch: %v", err)
			} else if hasNewCommits {
				isOutdated = true
				outdatedReason = fmt.Sprintf("%d new commit(s) since this branch diverged", mainCommitsSinceBranch)
			}
		}

		// Handle outdated base if detected
		if isOutdated {
			// Determine if we should fail or just warn
			failOnOutdated := cfg.FailOnOutdatedBase != nil && *cfg.FailOnOutdatedBase

			if failOnOutdated {
				return "", fmt.Errorf("the '%s' branch has %s. This branch is calculating versions based on an outdated '%s' branch. Rebase or merge from '%s' to continue", mainBranch, outdatedReason, mainBranch, mainBranch)
			} else {
				log("WARNING: The '%s' branch has %s.", mainBranch, outdatedReason)
				log("         This branch is calculating versions based on an outdated '%s' branch.", mainBranch)
				log("         Consider rebasing or merging from '%s' to get accurate version calculations.", mainBranch)
			}
		}

		// Calculate patch version: base + 1 (for the next version) + commits on main since branching
		if useTagAsBase {
			// Start with the next patch after the tag
			version.Patch = baseVersion.Patch + 1

			// Only add mainCommitsSinceBranch if the tag IS in the branch history
			// If the tag is NOT in branch history (e.g., added to main after branch diverged),
			// we don't add mainCommitsSinceBranch because the tag already represents the latest version
			if !tagNotInBranchHistory {
				version.Patch += mainCommitsSinceBranch
			}
		} else {
			// No tag base, use commit count (this maintains backward compatibility)
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

	// Apply mode conversion (which handles prefix internally for JSON mode)
	modeVersion, err := applyVersionMode(version.String(), cfg)
	if err != nil {
		return "", fmt.Errorf("failed to apply version mode: %w", err)
	}
	// For non-JSON modes, apply prefix here
	mode := defaults.DefaultMode
	if cfg.Mode != nil && *cfg.Mode != "" {
		mode = *cfg.Mode
	}
	if mode != defaults.ModeJson {
		result := applyVersionPrefix(modeVersion, cfg)
		if result != modeVersion {
			log("Applied version prefix: %s -> %s", modeVersion, result)
		}
		log("Final version: %s", result)
		return result, nil
	}
	log("Final version: %s", modeVersion)
	return modeVersion, nil
}

// applyVersionPrefix adds the configured version prefix to the version string
func applyVersionPrefix(version string, cfg *config.Config) string {
	if cfg.VersionPrefix != nil && *cfg.VersionPrefix != "" {
		return *cfg.VersionPrefix + version
	}
	return version
}

// applyVersionMode converts the version to the configured mode format
func applyVersionMode(version string, cfg *config.Config) (string, error) {
	mode := defaults.DefaultMode
	if cfg.Mode != nil && *cfg.Mode != "" {
		mode = *cfg.Mode
		log("Using configured version mode: %s", mode)
	} else {
		log("Using default version mode: %s", mode)
	}

	// Validate mode
	validMode := false
	for _, valid := range defaults.ValidModes {
		if mode == valid {
			validMode = true
			break
		}
	}
	if !validMode {
		return "", fmt.Errorf("invalid mode '%s': must be one of %v", mode, defaults.ValidModes)
	}

	// Apply mode conversion
	switch mode {
	case defaults.ModeJson:
		// Convert to PEP 440 format
		pep440Version, err := ConvertToPEP440(version)
		if err != nil {
			return "", fmt.Errorf("failed to convert to PEP 440: %w", err)
		}

		// Apply version prefix for the "WithPrefix" fields
		semverWithPrefix := applyVersionPrefix(version, cfg)
		pep440WithPrefix := applyVersionPrefix(pep440Version, cfg)

		// A version is a release if it has no prerelease identifier
		isRelease := !strings.Contains(version, "-")

		// Parse version to extract major, minor, patch
		parsedVersion, err := parseVersion(version)
		if err != nil {
			return "", fmt.Errorf("failed to parse version for JSON output: %w", err)
		}

		output := VersionOutput{
			Semver:           version,
			SemverWithPrefix: semverWithPrefix,
			Pep440:           pep440Version,
			Pep440WithPrefix: pep440WithPrefix,
			Major:            parsedVersion.Major,
			Minor:            parsedVersion.Minor,
			Patch:            parsedVersion.Patch,
			IsRelease:        isRelease,
		}

		jsonBytes, err := json.Marshal(output)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON output: %w", err)
		}

		log("Generated JSON output with semver=%s, semverWithPrefix=%s, pep440=%s, pep440WithPrefix=%s, major=%d, minor=%d, patch=%d, isRelease=%v",
			version, semverWithPrefix, pep440Version, pep440WithPrefix, parsedVersion.Major, parsedVersion.Minor, parsedVersion.Patch, isRelease)
		return string(jsonBytes), nil
	case defaults.ModePep440:
		pep440Version, err := ConvertToPEP440(version)
		if err != nil {
			return "", fmt.Errorf("failed to convert to PEP 440: %w", err)
		}
		if pep440Version != version {
			log("Converted to PEP 440 format: %s -> %s", version, pep440Version)
		}
		return pep440Version, nil
	case defaults.ModeSemver:
		// No conversion needed for semver
		return version, nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", mode)
	}
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
