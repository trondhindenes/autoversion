package version

import (
	"fmt"

	"github.com/trondhindenes/autoversion/internal/git"
)

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
	repo, err := git.OpenRepo(".")
	if err != nil {
		return "", err
	}

	// Check for tags first - tags take precedence over everything
	tag, err := repo.GetTagOnCurrentCommit()
	if err != nil {
		return "", err
	}

	if tag != "" {
		// Found a tag on the current commit
		version := git.StripTagPrefix(tag, tagPrefix)
		return version, nil
	}

	// No tag found, calculate version based on branch and commit count
	if mainBranch == "" {
		mainBranch = "main"
	}

	currentBranch, err := repo.GetCurrentBranch()
	if err != nil {
		return "", err
	}

	// Get commit count on main branch
	mainCommitCount, err := repo.GetMainBranchCommitCount(mainBranch)
	if err != nil {
		return "", err
	}

	version := Version{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}

	if currentBranch == mainBranch {
		// On main branch: version is 1.0.0, 1.0.1, 1.0.2, etc.
		commitCount, err := repo.GetCommitCount()
		if err != nil {
			return "", err
		}

		if commitCount > 0 {
			version.Patch = commitCount - 1
		}
	} else {
		// On feature branch: version is 1.0.X-branchname.Y
		// X is the next patch version (mainCommitCount)
		// Y is the number of commits on this branch since branching
		version.Patch = mainCommitCount

		branchCommitCount, err := repo.GetCommitCountSinceBranchPoint(mainBranch)
		if err != nil {
			return "", err
		}

		version.Prerelease = git.SanitizeBranchName(currentBranch)
		version.Build = branchCommitCount
	}

	return version.String(), nil
}
