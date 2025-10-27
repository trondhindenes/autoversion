# autoversion

A Go-based CLI tool that automatically generates semantic versions based on the state of a git repository.

## Features
- Generates unique semantic versions based on git commit history
- Pure semver output by default (e.g., `1.0.0`, but can be configured to add prefix, for example `v1.0.0`)
- Git tag support: tags on commits take precedence over calculated versions
- Configurable tag prefix stripping (e.g., `v2.0.0` → `2.0.0` or `PRODUCT/2.0.0` → `2.0.0`)
- Automatic main branch detection: Works with both `main` and `master` branches by default, can be configured
- Flexible main branch behavior:
  - `release` mode (default): Untagged main branch creates release versions like `1.0.0`, `1.0.1`, `1.0.2`, tags are optional
  - `pre` mode: Main branch creates prerelease versions like `1.0.0-pre.0`, `1.0.0-pre.1` (only tagged commits create releases)
- Feature branch prerelease versions: `1.0.2-feature.0`, `1.0.2-feature.1`, etc.
- CI/CD environment support with branch detection
- Supports both YAML and JSON configuration files
- JSON schema generation for configuration validation
- Zero configuration required - works with sensible defaults
- Pep440-compatible output option for python projects

## Requirements

- **Full git clone**: autoversion requires a full git history and will not work with shallow clones (created with `git clone --depth N`). If you have a shallow clone, convert it to a full clone with `git fetch --unshallow`.
- **GitHub Actions note**: autoversion automatically handles detached HEAD states in CI environments by checking both local and remote branch references.

## Installation

## Homebrew (macos only)
```
brew tap trondhindenes/autoversion
brew install autoversion
```

## Linux
We recommend using the excellent "bin" package manager (https://github.com/marcosnils/bin).
With that installed you can simply run:
```shell
bin install github.com/trondhindenes/autoversion
```

### GitHub Action (Recommended for CI/CD)

The easiest way to use autoversion in GitHub Actions:

```yaml
- name: Calculate version
  id: version
  uses: trondhindenes/autoversion-action@v1

- name: Use the version
  run: echo "Version is ${{ steps.version.outputs.version }}"
```

See [GitHub Action documentation](https://github.com/trondhindenes/autoversion-action) for complete usage examples.

### Docker

```bash
# Run in current directory
docker run --rm -v "$(pwd):/repo" ghcr.io/trondhindenes/autoversion:latest

# With custom config
docker run --rm -v "$(pwd):/repo" ghcr.io/trondhindenes/autoversion:latest --config .autoversion.yaml

# Generate schema
docker run --rm ghcr.io/trondhindenes/autoversion:latest schema
```
See https://github.com/trondhindenes/autoversion/pkgs/container/autoversion for available tags. 
The tag `latest` points to the newest release version and should be safe to use.

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



```bash
autoversion
```

Autoversion's output defaults to "json" mode, which will output a JSON object with the version, like (this example assumes that versionPrefix is set o "v"):
```json
  {
      "semver": "3.0.4",
      "semverWithPrefix": "v3.0.4",
      "pep440": "3.0.4",
      "pep440WithPrefix": "v3.0.4",  
      "major": 3,
      "minor": 0,
      "patch": 4,
      "isRelease": true
  }
```
Note that `semverWithPrefix` may contain a value that is not semver-compliant, and `pep440WithPrefix` may contain a value that is not pep440-compliant. This will happen if the `versionPrefix` setting is configured.
You can also set it to "semver" or "pep440" mode to get a pure semver or PEP 440 version respectively. In these modes, the `versionPrefix` is added to the calculated version.


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
mode: "json"                      # Default: "json" - or "semver" or "pep440"
tagPrefix: "v"                    # Default: "" (no stripping) - strips "v" from tags
versionPrefix: ""                 # Default: "" - set to "v" to add prefix to output
initialVersion: "1.0.0"           # Default: "1.0.0" - version to use when no tags exist
useCIBranch: true                 # Default: true - automatically detects branch in CI/CD environments
failOnOutdatedBase: false         # Default: false - set to true to fail instead of warn
outdatedBaseCheckMode: "tagged"   # Default: "tagged" - or "all" to check all commits
```

**JSON Example:**
```json
{
  "mainBranches": ["main", "master"],
  "mainBranchBehavior": "pre",
  "mode": "semver",
  "tagPrefix": "PRODUCT/",
  "versionPrefix": "v",
  "failOnOutdatedBase": true,
  "outdatedBaseCheckMode": "all"
}
```

### Custom Configuration File

You can specify a custom configuration file path:

```bash
autoversion --config /path/to/config.yaml
```

### Specify config inline
You can also specify the configuration inline:
```bash
autoversion --set-config "tagPrefix=v" --set-config "mainBranchBehavior=pre"
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

When running on the main branch (or configured primary branch) without a tag, and `mainBranchBehavior` has not been changed from the default:
- Version format: `MAJOR.MINOR.PATCH`
- Always starts at the configured `initialVersion` (which defaults to `1.0.0`)
- Each commit increments the PATCH version
- Example progression: `1.0.0` → `1.0.1` → `1.0.2`
- If `mainBranchBehavior` is set to `"pre"`, the version is a prerelease version (e.g., `1.0.0-pre.0`, `1.0.0-pre.1`, etc.)

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
| `mode` | string | `"json"` | Version output format mode: `"json"` (default) outputs JSON with all version formats, `"semver"` outputs standard semantic versioning, or `"pep440"` outputs Python PEP 440 compatible versions |
| `tagPrefix` | string | `""` (empty) | Prefix to strip from git tags (e.g., `"v"` strips `v2.0.0` → `2.0.0`, `"PRODUCT/"` strips `PRODUCT/2.0.0` → `2.0.0`) |
| `versionPrefix` | string | `""` (empty) | Prefix to add to the output version (e.g., `"v"` outputs `v1.0.0` instead of `1.0.0`). In JSON mode, this is included in the `semverWithPrefix` and `pep440WithPrefix` fields |
| `initialVersion` | string | `"1.0.0"` | The initial version to use when no tags exist in the repository (e.g., `"0.0.1"` or `"2.0.0"`). Must be valid semver |
| `useCIBranch` | boolean | `true` | Enable CI branch detection (useful for PR builds where CI checks out a detached HEAD). Automatically detects GitHub Actions, GitLab CI, CircleCI, Travis CI, Jenkins, and Azure Pipelines |
| `failOnOutdatedBase` | boolean | `false` | When running on a feature branch, if true and the main branch has been updated (based on `outdatedBaseCheckMode`) after this branch diverged, autoversion will exit with an error instead of just warning |
| `outdatedBaseCheckMode` | string | `"tagged"` | Controls what triggers the outdated base warning/error on feature branches: `"tagged"` (default) only warns when main has new tags, or `"all"` warns when main has any new commits since branching |

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
versionPrefix: "v"  # In JSON mode: adds prefix to semverWithPrefix and pep440WithPrefix fields
                    # In semver/pep440 modes: outputs v1.0.0 instead of 1.0.0
```

**Use semver mode instead of JSON:**
```yaml
# .autoversion.yaml
mode: "semver"  # Outputs pure semver string: 1.0.0 instead of JSON
```

**Use PEP 440 mode for Python projects:**
```yaml
# .autoversion.yaml
mode: "pep440"  # Outputs PEP 440 format: 3.0.4a0 for prereleases
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
useCIBranch: true  # Enabled by default - detects actual branch from CI environment variables
# Automatically detects: GitHub Actions, GitLab CI, CircleCI, Travis CI, Jenkins, Azure Pipelines
```

### Default Behavior Summary

When you run `autoversion` without any configuration file:
- Main branches are `main` or `master` (whichever exists, `main` preferred)
- Main branch behavior is `release` mode (creates release versions)
- Output mode is `json` (outputs JSON object with all version formats)
- Git tags are used as-is (no prefix stripping)
- Version prefix is empty (no prefix added to versions)
- Initial version (when no tags exist) is `1.0.0`
- Branch detection uses CI environment variables (enabled by default) when available (GitHub Actions, GitLab CI, etc.), falls back to git's current branch
- Outdated base check mode is `tagged` (warns only on new tags, not all commits)
- Fail on outdated base is `false` (warnings only, not errors)
- First commit on main outputs `{"semver":"1.0.0",...,"isRelease":true}`
- Each subsequent commit increments patch version (`1.0.1`, `1.0.2`, etc.)
- Feature branches output prerelease versions (`1.0.3-feature-name.0` with `isRelease:false`)

## Examples

### On main branch (default JSON mode):
```bash
$ git checkout main
$ autoversion
{"semver":"1.0.5","semverWithPrefix":"1.0.5","pep440":"1.0.5","pep440WithPrefix":"1.0.5","major":1,"minor":0,"patch":5,"isRelease":true}
```

### On feature branch (default JSON mode):
```bash
$ git checkout -b feature/new-widget
$ # make some commits
$ autoversion
{"semver":"1.0.6-new-widget.3","semverWithPrefix":"1.0.6-new-widget.3","pep440":"1.0.6a3","pep440WithPrefix":"1.0.6a3","major":1,"minor":0,"patch":6,"isRelease":false}
```

### Using semver mode:
```bash
$ autoversion --config-flag mode=semver
1.0.5
```


### With git tags (JSON mode):
```bash
# Without any configuration (tag returned as-is in JSON)
$ git tag -a v2.0.0 -m "Release 2.0.0"
$ autoversion
{"semver":"v2.0.0","semverWithPrefix":"v2.0.0","pep440":"v2.0.0","pep440WithPrefix":"v2.0.0","major":2,"minor":0,"patch":0,"isRelease":true}

# With tagPrefix: "v" configured (strips the "v")
$ autoversion
{"semver":"2.0.0","semverWithPrefix":"2.0.0","pep440":"2.0.0","pep440WithPrefix":"2.0.0","major":2,"minor":0,"patch":0,"isRelease":true}

# With tagPrefix: "v" AND versionPrefix: "v" configured
$ autoversion
{"semver":"2.0.0","semverWithPrefix":"v2.0.0","pep440":"2.0.0","pep440WithPrefix":"v2.0.0","major":2,"minor":0,"patch":0,"isRelease":true}
```

### With git tags (semver mode):
```bash
# With mode: "semver" and tagPrefix: "v" configured
$ git tag -a v2.0.0 -m "Release 2.0.0"
$ autoversion --config-flag mode=semver --config-flag tagPrefix=v
2.0.0
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

# Note that it's recommended to use the "official" github action instead of docker as shown below
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
