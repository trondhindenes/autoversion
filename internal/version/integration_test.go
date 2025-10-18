package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trondhindenes/autoversion/internal/config"
)

// TestIntegration runs comprehensive integration tests with real git repositories
func TestIntegration(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH, skipping integration tests")
	}

	t.Run("MainBranchVersioning", testMainBranchVersioning)
	t.Run("FeatureBranchVersioning", testFeatureBranchVersioning)
	t.Run("TagSupport", testTagSupport)
	t.Run("TagPrefixStripping", testTagPrefixStripping)
	t.Run("InvalidTagHandling", testInvalidTagHandling)
	t.Run("BranchSanitization", testBranchSanitization)
	t.Run("CIBranchDetection", testCIBranchDetection)
	t.Run("MultipleBranches", testMultipleBranches)
}

func testMainBranchVersioning(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	// First commit should be 1.0.0
	version, err := calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.0" {
		t.Errorf("Expected 1.0.0, got %s", version)
	}

	// Add more commits
	makeCommit(t, repo, "second commit")
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.1" {
		t.Errorf("Expected 1.0.1, got %s", version)
	}

	makeCommit(t, repo, "third commit")
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.2" {
		t.Errorf("Expected 1.0.2, got %s", version)
	}

	// Add several more commits
	for i := 0; i < 5; i++ {
		makeCommit(t, repo, "commit")
	}
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.7" {
		t.Errorf("Expected 1.0.7, got %s", version)
	}
}

func testFeatureBranchVersioning(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	// Add some commits to main
	makeCommit(t, repo, "second commit")
	makeCommit(t, repo, "third commit")

	// Main should be at 1.0.2
	version, err := calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.2" {
		t.Errorf("Expected 1.0.2, got %s", version)
	}

	// Create feature branch
	checkoutBranch(t, repo, "feature/new-widget", true)

	// First commit on feature branch should be 1.0.3-new-widget.0
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.3-new-widget.0" {
		t.Errorf("Expected 1.0.3-new-widget.0, got %s", version)
	}

	// Add commits to feature branch
	makeCommit(t, repo, "feature commit 1")
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.3-new-widget.1" {
		t.Errorf("Expected 1.0.3-new-widget.1, got %s", version)
	}

	makeCommit(t, repo, "feature commit 2")
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.3-new-widget.2" {
		t.Errorf("Expected 1.0.3-new-widget.2, got %s", version)
	}
}

func testTagSupport(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	// Add commits
	makeCommit(t, repo, "second commit")
	makeCommit(t, repo, "third commit")

	// Should be 1.0.2
	version, err := calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.2" {
		t.Errorf("Expected 1.0.2, got %s", version)
	}

	// Tag current commit
	createTag(t, repo, "2.0.0")

	// Should now return the tag
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "2.0.0" {
		t.Errorf("Expected 2.0.0 (from tag), got %s", version)
	}

	// Add another commit (no tag)
	makeCommit(t, repo, "fourth commit")

	// Should go back to calculated version
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.3" {
		t.Errorf("Expected 1.0.3, got %s", version)
	}
}

func testTagPrefixStripping(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	makeCommit(t, repo, "second commit")

	// Tag with v prefix
	createTag(t, repo, "v3.0.0")

	// Without tagPrefix stripping, "v3.0.0" is not valid semver, so it should be ignored
	// and fall back to calculated version (1.0.1 for 2nd commit)
	version, err := calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.1" {
		t.Errorf("Expected 1.0.1 (tag v3.0.0 is not valid semver), got %s", version)
	}

	// With tagPrefix "v", strips to "3.0.0" which IS valid semver
	version, err = calculateVersionInRepo(repo, "main", "v")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "3.0.0" {
		t.Errorf("Expected 3.0.0 (tagPrefix stripped), got %s", version)
	}

	// Test with PRODUCT/ prefix
	makeCommit(t, repo, "third commit")
	createTag(t, repo, "PRODUCT/4.1.0")

	version, err = calculateVersionInRepo(repo, "main", "PRODUCT/")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "4.1.0" {
		t.Errorf("Expected 4.1.0 (prefix stripped), got %s", version)
	}
}

func testInvalidTagHandling(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	makeCommit(t, repo, "second commit")
	makeCommit(t, repo, "third commit")

	// Create a tag that's not valid semver (even without prefix)
	createTag(t, repo, "release-2023")

	// Should ignore invalid tag and calculate version
	version, err := calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.2" {
		t.Errorf("Expected 1.0.2 (invalid tag ignored), got %s", version)
	}

	// Create another commit with a valid semver tag
	makeCommit(t, repo, "fourth commit")
	createTag(t, repo, "2.0.0")

	// Should use the valid tag
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "2.0.0" {
		t.Errorf("Expected 2.0.0 (valid tag), got %s", version)
	}
}

func testBranchSanitization(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	makeCommit(t, repo, "second commit")

	tests := []struct {
		branchName     string
		expectedPrefix string
	}{
		{"feature/add-login", "add-login"},
		{"bugfix/fix-crash", "fix-crash"},
		{"hotfix/security-patch", "security-patch"},
		{"feature/USER/new-feature", "user-new-feature"},
		{"my_custom_branch", "my-custom-branch"},
		{"FEATURE/TEST", "feature-test"},
	}

	for _, tt := range tests {
		t.Run(tt.branchName, func(t *testing.T) {
			// Switch back to main first
			checkoutBranch(t, repo, "main", false)

			// Create and checkout the test branch
			checkoutBranch(t, repo, tt.branchName, true)

			version, err := calculateVersionInRepo(repo, "main", "")
			if err != nil {
				t.Fatalf("Failed to calculate version: %v", err)
			}

			expectedVersion := "1.0.2-" + tt.expectedPrefix + ".0"
			if version != expectedVersion {
				t.Errorf("Expected %s, got %s", expectedVersion, version)
			}
		})
	}
}

func testCIBranchDetection(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	// Add commits to main
	makeCommit(t, repo, "second commit")
	makeCommit(t, repo, "third commit")

	// Create feature branch
	checkoutBranch(t, repo, "feature/ci-test", true)
	makeCommit(t, repo, "feature commit")

	// Now checkout main to simulate CI checking out a detached HEAD or temp branch
	checkoutBranch(t, repo, "main", false)

	// Set environment variable to simulate GitHub Actions
	os.Setenv("GITHUB_HEAD_REF", "feature/ci-test")
	defer os.Unsetenv("GITHUB_HEAD_REF")

	// Calculate version with CI branch detection enabled
	cfg := &config.Config{
		MainBranch:  "main",
		UseCIBranch: boolPtr(true),
	}

	// Change to repo directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("Failed to change to repo directory: %v", err)
	}
	defer os.Chdir(oldDir)

	version, err := CalculateWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}

	// Should detect the feature branch from environment variable
	if !strings.HasPrefix(version, "1.0.3-ci-test.") {
		t.Errorf("Expected version to start with 1.0.3-ci-test., got %s", version)
	}

	// Without CI branch detection, should use main branch
	cfg.UseCIBranch = boolPtr(false)
	version, err = CalculateWithConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}

	if version != "1.0.2" {
		t.Errorf("Expected 1.0.2 (main branch), got %s", version)
	}
}

func testMultipleBranches(t *testing.T) {
	repo := setupTestRepo(t, "main")
	defer cleanup(repo)

	// Build up main branch
	for i := 0; i < 5; i++ {
		makeCommit(t, repo, "main commit")
	}

	// Main should be at 1.0.5
	version, err := calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.5" {
		t.Errorf("Expected 1.0.5, got %s", version)
	}

	// Create first feature branch
	checkoutBranch(t, repo, "feature/branch-a", true)
	makeCommit(t, repo, "feature a commit 1")
	makeCommit(t, repo, "feature a commit 2")

	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.6-branch-a.2" {
		t.Errorf("Expected 1.0.6-branch-a.2, got %s", version)
	}

	// Go back to main and create another feature branch
	checkoutBranch(t, repo, "main", false)
	checkoutBranch(t, repo, "feature/branch-b", true)
	makeCommit(t, repo, "feature b commit 1")

	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.6-branch-b.1" {
		t.Errorf("Expected 1.0.6-branch-b.1, got %s", version)
	}

	// Switch back to branch-a, version should still be correct
	checkoutBranch(t, repo, "feature/branch-a", false)
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.6-branch-a.2" {
		t.Errorf("Expected 1.0.6-branch-a.2, got %s", version)
	}

	// Add more commits to main
	checkoutBranch(t, repo, "main", false)
	makeCommit(t, repo, "main commit")
	makeCommit(t, repo, "main commit")

	// Main should now be at 1.0.7
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.7" {
		t.Errorf("Expected 1.0.7, got %s", version)
	}

	// Feature branches should now show 1.0.8-... (next version)
	checkoutBranch(t, repo, "feature/branch-a", false)
	version, err = calculateVersionInRepo(repo, "main", "")
	if err != nil {
		t.Fatalf("Failed to calculate version: %v", err)
	}
	if version != "1.0.8-branch-a.2" {
		t.Errorf("Expected 1.0.8-branch-a.2, got %s", version)
	}
}

// Helper functions

func setupTestRepo(t *testing.T, mainBranch string) string {
	t.Helper()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "autoversion-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	runGit(t, tmpDir, "init", "-b", mainBranch)
	runGit(t, tmpDir, "config", "user.email", "test@example.com")
	runGit(t, tmpDir, "config", "user.name", "Test User")

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("initial content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	runGit(t, tmpDir, "add", "test.txt")
	runGit(t, tmpDir, "commit", "-m", "initial commit")

	return tmpDir
}

func cleanup(repoPath string) {
	os.RemoveAll(repoPath)
}

func makeCommit(t *testing.T, repoPath, message string) {
	t.Helper()

	testFile := filepath.Join(repoPath, "test.txt")
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Append to file to make a change
	newContent := string(content) + message + "\n"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	runGit(t, repoPath, "add", "test.txt")
	runGit(t, repoPath, "commit", "-m", message)
}

func checkoutBranch(t *testing.T, repoPath, branch string, create bool) {
	t.Helper()

	if create {
		runGit(t, repoPath, "checkout", "-b", branch)
	} else {
		runGit(t, repoPath, "checkout", branch)
	}
}

func createTag(t *testing.T, repoPath, tag string) {
	t.Helper()
	runGit(t, repoPath, "tag", "-a", tag, "-m", "Tag "+tag)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\nOutput: %s", args, err, output)
	}
}

func calculateVersionInRepo(repoPath, mainBranch, tagPrefix string) (string, error) {
	// Save current directory
	oldDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Change to repo directory
	if err := os.Chdir(repoPath); err != nil {
		return "", err
	}
	defer os.Chdir(oldDir)

	// Calculate version
	return Calculate(mainBranch, tagPrefix)
}

func boolPtr(b bool) *bool {
	return &b
}
