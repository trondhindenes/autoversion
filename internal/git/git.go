package git

import (
	"fmt"
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
func (g *Repo) GetMainBranchCommitCount(mainBranch string) (int, error) {
	refName := plumbing.NewBranchReferenceName(mainBranch)
	ref, err := g.repo.Reference(refName, true)
	if err != nil {
		return 0, fmt.Errorf("failed to get %s branch reference: %w", mainBranch, err)
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
func (g *Repo) GetCommitCountSinceBranchPoint(mainBranch string) (int, error) {
	currentBranch, err := g.GetCurrentBranch()
	if err != nil {
		return 0, err
	}

	if currentBranch == mainBranch {
		return 0, nil
	}

	head, err := g.repo.Head()
	if err != nil {
		return 0, fmt.Errorf("failed to get HEAD: %w", err)
	}

	mainRefName := plumbing.NewBranchReferenceName(mainBranch)
	mainRef, err := g.repo.Reference(mainRefName, true)
	if err != nil {
		return 0, fmt.Errorf("failed to get %s branch reference: %w", mainBranch, err)
	}

	// Get commits on current branch
	currentCommits := make(map[plumbing.Hash]bool)
	commitIter, err := g.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return 0, fmt.Errorf("failed to get current branch log: %w", err)
	}

	err = commitIter.ForEach(func(c *object.Commit) error {
		currentCommits[c.Hash] = true
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("failed to iterate current branch commits: %w", err)
	}

	// Find common ancestor by checking main branch commits
	mainCommitIter, err := g.repo.Log(&git.LogOptions{From: mainRef.Hash()})
	if err != nil {
		return 0, fmt.Errorf("failed to get main branch log: %w", err)
	}

	var commonAncestor plumbing.Hash
	err = mainCommitIter.ForEach(func(c *object.Commit) error {
		if currentCommits[c.Hash] {
			commonAncestor = c.Hash
			return storer.ErrStop
		}
		return nil
	})
	if err != nil && err != storer.ErrStop {
		return 0, fmt.Errorf("failed to find common ancestor: %w", err)
	}

	// Count commits since common ancestor
	count := 0
	commitIter2, err := g.repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return 0, fmt.Errorf("failed to get commit log: %w", err)
	}

	err = commitIter2.ForEach(func(c *object.Commit) error {
		if c.Hash == commonAncestor {
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
