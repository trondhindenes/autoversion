#!/bin/bash
#
# Install Git hooks for the autoversion project
#
# This script installs pre-commit hooks that automatically format Go code.
# Run this after cloning the repository to set up your development environment.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_SRC="$SCRIPT_DIR/git-hooks"
HOOKS_DEST="$REPO_ROOT/.git/hooks"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Installing Git hooks...${NC}"

# Check if .git directory exists
if [ ! -d "$REPO_ROOT/.git" ]; then
    echo "Error: .git directory not found. Are you in a Git repository?"
    exit 1
fi

# Install pre-commit hook
if [ -f "$HOOKS_SRC/pre-commit" ]; then
    cp "$HOOKS_SRC/pre-commit" "$HOOKS_DEST/pre-commit"
    chmod +x "$HOOKS_DEST/pre-commit"
    echo -e "${GREEN}✓${NC} Installed pre-commit hook (auto-formats Go code)"
else
    echo "Warning: pre-commit hook template not found"
fi

echo -e "${GREEN}Git hooks installed successfully!${NC}"
echo ""
echo "The pre-commit hook will automatically format Go code using gofmt before each commit."
echo "To skip the hook for a specific commit, use: git commit --no-verify"
