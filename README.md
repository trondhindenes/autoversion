# autoversion

A Go-based CLI tool that automatically generates semantic versions based on the state of a git repository.

## Features

- **Pure semver output by default** (e.g., `1.0.0`, not `v1.0.0`)
- Generates semantic versions based on git commit history
- Git tag support: tags on commits take precedence over calculated versions
- Configurable tag prefix stripping (e.g., `v2.0.0` → `2.0.0` or `PRODUCT/2.0.0` → `2.0.0`)
- Configurable version prefix for output (e.g., add `v` to output `v1.0.0`)
- **Automatic main branch detection**: Works with both `main` and `master` branches by default
- **Flexible main branch behavior**:
  - `release` mode (default): Main branch creates release versions like `1.0.0`, `1.0.1`, `1.0.2`
  - `pre` mode: Main branch creates prerelease versions like `1.0.0-pre.0`, `1.0.0-pre.1` (only tagged commits create releases)
- Feature branch prerelease versions: `1.0.2-feature.0`, `1.0.2-feature.1`, etc.
- CI/CD environment support with branch detection
- Supports both YAML and JSON configuration files
- JSON schema generation for configuration validation
- Zero configuration required - works with sensible defaults

## Requirements

- **Full git clone**: autoversion requires a full git history and will not work with shallow clones (created with `git clone --depth N`). If you have a shallow clone, convert it to a full clone with `git fetch --unshallow`.
- **GitHub Actions note**: autoversion automatically handles detached HEAD states in CI environments by checking both local and remote branch references.

## Installation

### GitHub Action (Recommended for CI/CD)

The easiest way to use autoversion in GitHub Actions:

```yaml
- name: Calculate version
  id: version
  uses: trondhindenes/autoversion@v1

- name: Use the version
  run: echo "Version is ${{ steps.version.outputs.version }}"
```

See [GitHub Action documentation](ACTION.md) for complete usage examples.

### Docker

The easiest way to use autoversion is via Docker:

```bash
# Run in current directory
docker run --rm -v "$(pwd):/repo" ghcr.io/trondhindenes/autoversion:latest

# With custom config
docker run --rm -v "$(pwd):/repo" ghcr.io/trondhindenes/autoversion:latest --config .autoversion.yaml

# Generate schema
docker run --rm ghcr.io/trondhindenes/autoversion:latest schema
```

**Available tags:**
- `latest` - Latest stable release
- `v1`, `v1.0`, `v1.0.13` - Specific versions with semver precision

### Download Pre-built Binaries

Download the latest release for your platform from the [releases page](https://github.com/trondhindenes/autoversion/releases).

Binaries are available for:
- Linux (amd64, arm64, armv7)
- macOS (amd64, arm64)
- Windows (amd64, arm64)
- FreeBSD (amd64, arm64)

**Naming Convention:**
- Linux/macOS/FreeBSD: `autoversion-{os}-{arch}.tar.gz`
- Windows: `autoversion-{os}-{arch}.zip`

**Quick Install (Linux/macOS):**
```bash
# Example for Linux amd64
curl -L https://github.com/trondhindenes/autoversion/releases/latest/download/autoversion-linux-amd64.tar.gz | tar xz
sudo mv autoversion /usr/local/bin/
```

### Install via Go

```bash
go install github.com/trondhindenes/autoversion/cmd/autoversion@latest
```

### Build from Source

```bash
git clone https://github.com/trondhindenes/autoversion.git
cd autoversion
make build
```

The binary will be placed in the `bin/` directory.

## Usage

### Basic Usage

Run autoversion in a git repository to get the current version:

```bash
autoversion
```

This will output a semantic version like:
- `1.0.0` - First commit on main branch
- `1.0.5` - Sixth commit on main branch
- `1.0.6-feature.0` - First commit on a feature branch when main is at 1.0.5

### Configuration

Create an optional configuration file named `.autoversion.yaml` or `.autoversion.json` in your repository root.

**Note:** All configuration options are optional with sensible defaults (see Configuration Options section below).

**YAML Example:**
```yaml
mainBranches: ["main", "master"]  # Default: ["main", "master"]
mainBranchBehavior: "release"     # Default: "release" - or "pre" for prerelease versions
tagPrefix: "v"                    # Default: "" (no stripping) - strips "v" from tags
versionPrefix: ""                 # Default: "" - set to "v" to output v1.0.0
initialVersion: "1.0.0"           # Default: "1.0.0" - version to use when no tags exist
useCIBranch: false               # Default: false - enable for CI/CD environments
```

**JSON Example:**
```json
{
  "mainBranches": ["main", "master"],
  "mainBranchBehavior": "pre",
  "tagPrefix": "PRODUCT/",
  "versionPrefix": ""
}
```

### Custom Configuration File

You can specify a custom configuration file path:

```bash
autoversion --config /path/to/config.yaml
```

### Generate Configuration Schema

Generate a JSON schema for the configuration file:

```bash
autoversion schema
```

This outputs a JSON schema that can be used for IDE autocompletion and validation.

## How It Works

### Version Priority

autoversion determines the version using the following priority order:

1. **Git Tags** (highest priority): If the current commit has a tag, that tag is used as the version (after stripping any configured prefix)
2. **Main Branch**: Calculated version based on commit count
3. **Feature Branch**: Prerelease version based on branch name and commit count

### Git Tag Versioning

When the current commit has a git tag:
- The tag name is used as the base version
- If `tagPrefix` is configured, it's stripped from the tag name first
- If `versionPrefix` is configured, it's added to the output
- Tags take precedence regardless of branch

**Examples:**
- No config: tag `v2.0.0` → output `v2.0.0`
- `tagPrefix: "v"`: tag `v2.0.0` → output `2.0.0`
- `tagPrefix: "PRODUCT/"`: tag `PRODUCT/3.1.0` → output `3.1.0`
- `tagPrefix: "v"` + `versionPrefix: "v"`: tag `v2.0.0` → output `v2.0.0`

### Main Branch Versioning

When running on the main branch (or configured primary branch) without a tag:
- Version format: `MAJOR.MINOR.PATCH`
- Always starts at `1.0.0`
- Each commit increments the PATCH version
- Example progression: `1.0.0` → `1.0.1` → `1.0.2`

### Feature Branch Versioning

When running on a non-main branch without a tag:
- Version format: `MAJOR.MINOR.PATCH-PRERELEASE.BUILD`
- PATCH is set to the next version after the main branch's current version
- PRERELEASE is derived from the branch name (sanitized)
- BUILD is the number of commits on the branch since it diverged from main
- Example: `1.0.2-add-new-feature.3`

### Branch Name Sanitization

Branch names are automatically sanitized for semver compatibility:
- Common prefixes removed (`feature/`, `bugfix/`, `hotfix/`, `release/`)
- Invalid characters replaced with hyphens
- Converted to lowercase
- Multiple consecutive hyphens collapsed

Examples:
- `feature/add-new-feature` → `add-new-feature`
- `bugfix/fix-crash` → `fix-crash`
- `feature/USER/login` → `user-login`

## Configuration Options

All configuration options are optional. If not specified, the defaults shown below will be used.

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mainBranches` | array | `["main", "master"]` | List of branch names to treat as main branches. The first matching branch found in the repository is used |
| `mainBranchBehavior` | string | `"release"` | Behavior for non-tagged commits on main branch: `"release"` creates release versions (`1.0.0`, `1.0.1`) or `"pre"` creates prerelease versions (`1.0.0-pre.0`, `1.0.0-pre.1`). Tagged commits always create release versions |
| `mainBranch` | string | (deprecated) | Deprecated: Use `mainBranches` instead. Still supported for backward compatibility |
| `tagPrefix` | string | `""` (empty) | Prefix to strip from git tags (e.g., `"v"` strips `v2.0.0` → `2.0.0`, `"PRODUCT/"` strips `PRODUCT/2.0.0` → `2.0.0`) |
| `versionPrefix` | string | `""` (empty) | Prefix to add to the output version (e.g., `"v"` outputs `v1.0.0` instead of `1.0.0`) |
| `initialVersion` | string | `"1.0.0"` | The initial version to use when no tags exist in the repository (e.g., `"0.0.1"` or `"2.0.0"`). Must be valid semver |
| `useCIBranch` | boolean | `false` | Enable CI branch detection (useful for PR builds where CI checks out a detached HEAD) |
| `ciProviders` | object | `{}` (empty) | Custom CI provider configurations for branch detection |

### Configuration Examples

**Minimal configuration (all defaults):**
```yaml
# .autoversion.yaml
# Empty file - all defaults will be used
```

**Common configuration (strip 'v' from tags, keep pure semver output):**
```yaml
# .autoversion.yaml
tagPrefix: "v"  # Strips "v" prefix from tags (v2.0.0 → 2.0.0)
# mainBranches not set - will use default ["main", "master"]
# versionPrefix not set - output will be pure semver (1.0.0)
```

**Prerelease mode (only tagged commits create releases):**
```yaml
# .autoversion.yaml
mainBranchBehavior: "pre"
# Non-tagged commits on main will be: 1.0.0-pre.0, 1.0.0-pre.1, etc.
# Tagged commits will be: 1.0.0, 2.0.0, etc.
```

**Output version with 'v' prefix:**
```yaml
# .autoversion.yaml
versionPrefix: "v"  # Outputs v1.0.0 instead of 1.0.0
```

**Product with custom tag prefix:**
```yaml
# .autoversion.yaml
mainBranch: main
tagPrefix: "PRODUCT/"  # Strips PRODUCT/ from tags
```

**Start versioning from 0.0.1:**
```yaml
# .autoversion.yaml
initialVersion: "0.0.1"  # First commit outputs 0.0.1 instead of 1.0.0
```

**Custom main branch names:**
```yaml
# .autoversion.yaml
mainBranches: ["trunk", "mainline"]
# Only these branches will be treated as main branches
```

**CI/CD environment (GitHub Actions, GitLab CI, etc.):**
```yaml
# .autoversion.yaml
useCIBranch: true  # Detect actual branch from CI environment variables
```

**Custom CI provider:**
```yaml
# .autoversion.yaml
mainBranch: main
useCIBranch: true
ciProviders:
  my-ci-system:
    branchEnvVar: "MY_CI_BRANCH_NAME"
```

### Default Behavior Summary

When you run `autoversion` without any configuration file:
- Main branches are `main` or `master` (whichever exists, `main` preferred)
- Main branch behavior is `release` mode (creates release versions)
- Git tags are used as-is (no prefix stripping)
- Output is pure semver format (`1.0.0`, not `v1.0.0`)
- Initial version (when no tags exist) is `1.0.0`
- Branch detection uses git's current branch (no CI environment detection)
- First commit on main outputs `1.0.0`
- Each subsequent commit increments patch version (`1.0.1`, `1.0.2`, etc.)
- Feature branches output prerelease versions (`1.0.3-feature-name.0`)

## Examples

### On main branch:
```bash
$ git checkout main
$ autoversion
1.0.5
```

### On feature branch:
```bash
$ git checkout -b feature/new-widget
$ # make some commits
$ autoversion
1.0.6-new-widget.3
```

### With custom config:
```bash
$ autoversion --config .autoversion.json
Using config file: .autoversion.json
1.0.2
```

### With git tags:
```bash
# Without any configuration (tag returned as-is)
$ git tag -a v2.0.0 -m "Release 2.0.0"
$ autoversion
v2.0.0

# With tagPrefix: "v" configured (strips the "v")
$ autoversion
2.0.0

# With tagPrefix: "v" AND versionPrefix: "v" configured
$ autoversion
v2.0.0
```

### With prerelease mode on main branch:
```yaml
# .autoversion.yaml
mainBranchBehavior: "pre"
```

```bash
# Non-tagged commits create prereleases
$ git checkout main
$ # make some commits
$ autoversion
1.0.0-pre.2

# Tagged commits create releases
$ git tag -a "1.0.0" -m "Release 1.0.0"
$ autoversion
1.0.0

# Commits after tags continue as prereleases
$ # make another commit
$ autoversion
1.0.1-pre.0
```

### With tag prefix (e.g., PRODUCT/):
```yaml
# .autoversion.yaml
tagPrefix: "PRODUCT/"
```

```bash
$ git tag -a PRODUCT/3.1.0 -m "Product Release 3.1.0"
$ autoversion
3.1.0
```

### Using Docker in CI/CD:

**Important**: Many CI systems use shallow clones by default for performance. You must configure your CI to use a full clone for autoversion to work correctly.

**GitHub Actions:**
```yaml
- name: Checkout code
  uses: actions/checkout@v4
  with:
    fetch-depth: 0  # Required: fetch full history for autoversion

- name: Get version
  id: version
  run: |
    VERSION=$(docker run --rm -v "${{ github.workspace }}:/repo" ghcr.io/trondhindenes/autoversion:latest)
    echo "version=${VERSION}" >> $GITHUB_OUTPUT
```

**GitLab CI:**
```yaml
get-version:
  image: ghcr.io/trondhindenes/autoversion:latest
  variables:
    GIT_DEPTH: 0  # Required: fetch full history for autoversion
  script:
    - autoversion > version.txt
  artifacts:
    paths:
      - version.txt
```

**Generic CI with Docker:**
```bash
# Capture version
VERSION=$(docker run --rm -v "$(pwd):/repo" ghcr.io/trondhindenes/autoversion:latest 2>/dev/null)
echo "Version: $VERSION"

# Use in build
docker build -t myapp:${VERSION} .
```

## Development

### Setup Git Hooks (Recommended)

After cloning the repository, install the Git hooks to automatically format code before commits:

```bash
./scripts/install-git-hooks.sh
```

This installs a pre-commit hook that:
- Automatically runs `gofmt -w -s` on all staged Go files
- Re-stages the formatted files
- Shows which files were formatted

The hook ensures all committed code is properly formatted.

**Skip the hook for a specific commit:**
```bash
git commit --no-verify
```

### Using the Makefile

```bash
# Build the project (output to bin/)
make build

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run go vet
make vet

# Clean build artifacts
make clean

# See all available targets
make help
```

### Manual Commands

```bash
# Run tests
go test -v ./...

# Build manually
go build -o bin/autoversion ./cmd/autoversion

# Format code
gofmt -w -s .
```
