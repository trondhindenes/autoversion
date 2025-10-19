package git

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Repo represents a git repository
type Repo struct {
	repo *git.Repository
}

// OpenRepo opens a git repository at the given path
func OpenRepo(path string) (*Repo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	repo, err := git.PlainOpen(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	return &Repo{repo: repo}, nil
}

// IsShallow checks if the repository is a shallow clone
func (g *Repo) IsShallow() (bool, error) {
	// Get the repository's worktree to access the git directory
	worktree, err := g.repo.Worktree()
	if err != nil {
		return false, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Construct path to .git/shallow file
	gitDir := filepath.Join(worktree.Filesystem.Root(), ".git")

	// Check if .git is a directory or a file (for worktrees)
	info, err := os.Stat(gitDir)
	if err != nil {
		return false, fmt.Errorf("failed to stat .git: %w", err)
	}

	var shallowPath string
	if info.IsDir() {
		// Normal repository
		shallowPath = filepath.Join(gitDir, "shallow")
	} else {
		// Worktree - need to read .git file to find actual git dir
		// For simplicity, we'll use the storer to check for shallow
		// go-git stores this information in the storer
		// We can check if there are any shallow commits
		shallows, err := g.repo.Storer.Shallow()
		if err != nil {
			return false, fmt.Errorf("failed to check shallow status: %w", err)
		}
		return len(shallows) > 0, nil
	}

	// Check if shallow file exists
	_, err = os.Stat(shallowPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check shallow file: %w", err)
	}

	return true, nil
}

// GetCurrentBranch returns the name of the current branch
func (g *Repo) GetCurrentBranch() (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if !head.Name().IsBranch() {
		return "", fmt.Errorf("HEAD is not pointing to a branch")
	}

	return head.Name().Short(), nil
}

// IsMainBranch checks if the given branch name matches any of the main branches
func IsMainBranch(currentBranch string, mainBranches []string) bool {
	for _, mainBranch := range mainBranches {
		if currentBranch == mainBranch {
			return true
		}
	}
	return false
}

// GetMainBranch returns the first main branch that exists in the repository
// It checks both local and remote branches to handle detached HEAD states in CI
func (g *Repo) GetMainBranch(mainBranches []string) (string, error) {
	for _, branchName := range mainBranches {
		// Try local branch first
		branchRefName := plumbing.NewBranchReferenceName(branchName)
		_, err := g.repo.Reference(branchRefName, true)
		if err == nil {
			return branchName, nil
		}

		// Try remote branch (e.g., origin/main, origin/master)
		remoteBranchRefName := plumbing.NewRemoteReferenceName("origin", branchName)
		_, err = g.repo.Reference(remoteBranchRefName, true)
		if err == nil {
			return branchName, nil
		}
	}
	return "", fmt.Errorf("none of the configured main branches exist: %v", mainBranches)
}

// GetCommitCount returns the number of commits on the current branch
func (g *Repo) GetCommitCount() (int, error) {
	head, err := g.repo.Head()
	if err != nil {
		return 0, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}

	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return count, nil
}

// GetMainBranchCommitCount returns the commit count on the main branch
// It checks both local and remote branches to handle detached HEAD states in CI
func (g *Repo) GetMainBranchCommitCount(mainBranch string) (int, error) {
	// Try local branch first
	refName := plumbing.NewBranchReferenceName(mainBranch)
	ref, err := g.repo.Reference(refName, true)

	// If local branch doesn't exist, try remote branch
	if err != nil {
		remoteBranchRefName := plumbing.NewRemoteReferenceName("origin", mainBranch)
		ref, err = g.repo.Reference(remoteBranchRefName, true)
		if err != nil {
			return 0, fmt.Errorf("failed to get %s branch reference (tried both local and remote): %w", mainBranch, err)
		}
	}

	commitIter, err := g.repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}

	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return count, nil
}

// GetCommitCountSinceBranchPoint returns the number of commits since branching from main
// This uses a proper merge-base algorithm to find the common ancestor
func (g *Repo) GetCommitCountSinceBranchPoint(mainBranch, currentBranch string) (int, error) {
	if currentBranch == mainBranch {
		return 0, nil
	}

	head, err := g.repo.Head()
	if err != nil {
		return 0, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Try local branch first
	mainRefName := plumbing.NewBranchReferenceName(mainBranch)
	mainRef, err := g.repo.Reference(mainRefName, true)

	// If local branch doesn't exist, try remote branch
	if err != nil {
		remoteBranchRefName := plumbing.NewRemoteReferenceName("origin", mainBranch)
		mainRef, err = g.repo.Reference(remoteBranchRefName, true)
		if err != nil {
			return 0, fmt.Errorf("failed to get %s branch reference (tried both local and remote): %w", mainBranch, err)
		}
	}

	// Find merge base (common ancestor) between current HEAD and main branch
	// This properly handles cases where main has moved forward after the branch was created
	mergeBase, err := g.findMergeBase(head.Hash(), mainRef.Hash())
	if err != nil {
		return 0, fmt.Errorf("failed to find merge base: %w", err)
	}

	// Count commits from HEAD back to merge base
	count := 0
	commitIter, err := g.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}

	err = commitIter.ForEach(func(c *object.Commit) error {
		if c.Hash == mergeBase {
			return storer.ErrStop
		}
		count++
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return 0, fmt.Errorf("failed to count commits since branch point: %w", err)
	}

	return count, nil
}

// findMergeBase finds the best common ancestor between two commits
// This implements a simplified version of git merge-base
func (g *Repo) findMergeBase(commit1Hash, commit2Hash plumbing.Hash) (plumbing.Hash, error) {
	// Get all ancestors of commit1
	ancestors1 := make(map[plumbing.Hash]int)
	distance := 0

	commitIter, err := g.repo.Log(&git.LogOptions{From: commit1Hash})
	if err != nil {
		return plumbing.ZeroHash, err
	}

	err = commitIter.ForEach(func(c *object.Commit) error {
		ancestors1[c.Hash] = distance
		distance++
		return nil
	})
	if err != nil {
		return plumbing.ZeroHash, err
	}

	// Walk commit2's history until we find a commit that's also in commit1's history
	// This is the merge base
	commitIter2, err := g.repo.Log(&git.LogOptions{From: commit2Hash})
	if err != nil {
		return plumbing.ZeroHash, err
	}

	var mergeBase plumbing.Hash
	err = commitIter2.ForEach(func(c *object.Commit) error {
		if _, exists := ancestors1[c.Hash]; exists {
			mergeBase = c.Hash
			return storer.ErrStop
		}
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return plumbing.ZeroHash, err
	}

	if mergeBase.IsZero() {
		return plumbing.ZeroHash, fmt.Errorf("no common ancestor found")
	}

	return mergeBase, nil
}

// GetTagOnCurrentCommit returns the tag on the current HEAD commit, if any
func (g *Repo) GetTagOnCurrentCommit() (string, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	headHash := head.Hash()

	// Iterate through all tags
	tagRefs, err := g.repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	var foundTag string
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		// Check if this tag points to the current commit
		if ref.Hash() == headHash {
			foundTag = ref.Name().Short()
			return storer.ErrStop
		}

		// Check if it's an annotated tag
		tag, err := g.repo.TagObject(ref.Hash())
		if err == nil {
			if tag.Target == headHash {
				foundTag = ref.Name().Short()
				return storer.ErrStop
			}
		}

		return nil
	})

	if err != nil && err != storer.ErrStop {
		return "", fmt.Errorf("failed to iterate tags: %w", err)
	}

	return foundTag, nil
}

// GetMostRecentTag returns the most recent tag in the commit history (walking back from HEAD)
// Returns the tag name and the number of commits since that tag
func (g *Repo) GetMostRecentTag() (string, int, error) {
	head, err := g.repo.Head()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get all tags
	tagRefs, err := g.repo.Tags()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get tags: %w", err)
	}

	// Build a map of commit hash to tag name
	tagMap := make(map[plumbing.Hash]string)
	err = tagRefs.ForEach(func(ref *plumbing.Reference) error {
		tagMap[ref.Hash()] = ref.Name().Short()

		// Also check annotated tags
		tag, err := g.repo.TagObject(ref.Hash())
		if err == nil {
			tagMap[tag.Target] = ref.Name().Short()
		}

		return nil
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to iterate tags: %w", err)
	}

	// Walk the commit history from HEAD
	commitIter, err := g.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return "", 0, fmt.Errorf("failed to get commit log: %w", err)
	}

	commitsSinceTag := 0
	var foundTag string
	err = commitIter.ForEach(func(c *object.Commit) error {
		if tagName, exists := tagMap[c.Hash]; exists {
			foundTag = tagName
			return storer.ErrStop
		}
		commitsSinceTag++
		return nil
	})

	if err != nil && err != storer.ErrStop {
		return "", 0, fmt.Errorf("failed to iterate commits: %w", err)
	}

	return foundTag, commitsSinceTag, nil
}

// StripTagPrefix removes the configured prefix from a tag name
func StripTagPrefix(tag, prefix string) string {
	if prefix == "" {
		return tag
	}
	return strings.TrimPrefix(tag, prefix)
}

// SanitizeBranchName converts a branch name to a valid prerelease identifier
func SanitizeBranchName(branch string) string {
	// Remove common prefixes
	branch = strings.TrimPrefix(branch, "feature/")
	branch = strings.TrimPrefix(branch, "bugfix/")
	branch = strings.TrimPrefix(branch, "hotfix/")
	branch = strings.TrimPrefix(branch, "release/")

	// Replace invalid characters with hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9-]`)
	branch = reg.ReplaceAllString(branch, "-")

	// Remove leading/trailing hyphens
	branch = strings.Trim(branch, "-")

	// Collapse multiple hyphens
	reg2 := regexp.MustCompile(`-+`)
	branch = reg2.ReplaceAllString(branch, "-")

	// Convert to lowercase
	branch = strings.ToLower(branch)

	if branch == "" {
		branch = "unknown"
	}

	return branch
}
