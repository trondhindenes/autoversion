# Autoversion GitHub Action

Automatically generate semantic versions in your GitHub Actions workflows based on git repository state.

## Features

- 🚀 Automatic version calculation based on commits and tags
- ⚡ Fast execution using pre-built Docker images
- 🏷️ Support for both `main` and `master` branches
- 🔀 Feature branch prerelease versions
- 🎯 Multiple output formats (full version, major, minor, patch)
- ⚙️ Configurable via `.autoversion.yaml`
- 📦 No external dependencies required

## Usage

### Basic Example

```yaml
name: Build and Release

on:
  push:
    branches:
      - main
      - 'feature/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required: fetch full history

      - name: Calculate version
        id: version
        uses: trondhindenes/autoversion@v1

      - name: Build with version
        run: |
          echo "Building version ${{ steps.version.outputs.version }}"
          # Your build commands here
```

### With Custom Configuration

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
    with:
      fetch-depth: 0  # Required: fetch full history

  - name: Calculate version
    id: version
    uses: trondhindenes/autoversion@v1
    with:
      config: '.autoversion.yaml'

  - name: Use version outputs
    run: |
      echo "Version: ${{ steps.version.outputs.version }}"
      echo "Major: ${{ steps.version.outputs.major }}"
      echo "Minor: ${{ steps.version.outputs.minor }}"
      echo "Patch: ${{ steps.version.outputs.patch }}"
      echo "Is prerelease: ${{ steps.version.outputs.is-prerelease }}"
```

### Pin to Specific Autoversion Version

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
    with:
      fetch-depth: 0

  - name: Calculate version with specific autoversion version
    id: version
    uses: trondhindenes/autoversion@v1
    with:
      version: '1.0.5'  # Use a specific version of autoversion
```

### Create Git Tag and Release

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
    with:
      fetch-depth: 0

  - name: Calculate version
    id: version
    uses: trondhindenes/autoversion@v1

  - name: Create tag
    if: github.ref == 'refs/heads/main'
    run: |
      git config user.name "github-actions[bot]"
      git config user.email "github-actions[bot]@users.noreply.github.com"
      git tag -a "v${{ steps.version.outputs.version }}" -m "Release v${{ steps.version.outputs.version }}"
      git push origin "v${{ steps.version.outputs.version }}"

  - name: Create GitHub Release
    if: github.ref == 'refs/heads/main'
    uses: softprops/action-gh-release@v1
    with:
      tag_name: v${{ steps.version.outputs.version }}
      generate_release_notes: true
```

### Build Docker Image with Version

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
    with:
      fetch-depth: 0

  - name: Calculate version
    id: version
    uses: trondhindenes/autoversion@v1

  - name: Set up Docker Buildx
    uses: docker/setup-buildx-action@v3

  - name: Build and push
    uses: docker/build-push-action@v5
    with:
      context: .
      push: true
      tags: |
        myorg/myapp:${{ steps.version.outputs.version }}
        myorg/myapp:${{ steps.version.outputs.major }}.${{ steps.version.outputs.minor }}
        myorg/myapp:${{ steps.version.outputs.major }}
        myorg/myapp:latest
```

### Conditional Logic Based on Prerelease

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v4
    with:
      fetch-depth: 0

  - name: Calculate version
    id: version
    uses: trondhindenes/autoversion@v1

  - name: Deploy to production
    if: steps.version.outputs.is-prerelease == 'false'
    run: |
      echo "Deploying release version ${{ steps.version.outputs.version }} to production"
      # Your deployment commands

  - name: Deploy to staging
    if: steps.version.outputs.is-prerelease == 'true'
    run: |
      echo "Deploying prerelease version ${{ steps.version.outputs.version }} to staging"
      # Your deployment commands
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `config` | Path to configuration file | No | `.autoversion.yaml` |
| `fail-on-error` | Whether to fail the action if version calculation fails | No | `true` |
| `version` | Specific version of autoversion to use (e.g., `"1"`, `"1.0"`, `"1.0.0"`, or `"latest"`) | No | `"1"` |

## Outputs

| Output | Description | Example |
|--------|-------------|---------|
| `version` | The full semantic version | `1.0.5` or `1.0.6-feature.3` |
| `major` | The major version number | `1` |
| `minor` | The minor version number | `0` |
| `patch` | The patch version number | `5` |
| `prerelease` | The prerelease identifier (empty if release) | `feature.3` or `pre.0` |
| `is-prerelease` | Whether this is a prerelease version | `true` or `false` |

## Configuration

Create a `.autoversion.yaml` file in your repository root to customize behavior:

```yaml
# Main branches (defaults to ["main", "master"])
mainBranches: ["main", "master"]

# Main branch behavior: "release" or "pre"
# - release (default): Creates release versions (1.0.0, 1.0.1)
# - pre: Creates prerelease versions (1.0.0-pre.0, 1.0.0-pre.1)
mainBranchBehavior: "release"

# Strip prefix from tags (e.g., "v" strips v1.0.0 → 1.0.0)
tagPrefix: "v"

# Add prefix to output version
versionPrefix: ""

# Initial version when no tags exist
initialVersion: "1.0.0"
```

See the [main README](README.md) for full configuration documentation.

## Important Requirements

### Fetch Full History

**You must use `fetch-depth: 0` when checking out your repository**, otherwise autoversion will fail because it needs the full git history to calculate versions correctly:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0  # This is required!
```

### Docker Availability

The GitHub Action runs autoversion in a Docker container, so **Docker must be available** in your runner environment. All GitHub-hosted runners (ubuntu-latest, etc.) include Docker by default. If using self-hosted runners, ensure Docker is installed and accessible.

## How It Works

### Main Branch (Release Mode)
```
Commit 1 → 1.0.0
Commit 2 → 1.0.1
Tag v2.0.0 → 2.0.0
Commit 3 → 2.0.1
```

### Main Branch (Pre Mode)
```
Commit 1 → 1.0.0-pre.0
Commit 2 → 1.0.0-pre.1
Tag 1.0.0 → 1.0.0
Commit 3 → 1.0.1-pre.0
```

### Feature Branch
```
Branch from main (at 1.0.5)
Feature commit 1 → 1.0.6-feature-name.0
Feature commit 2 → 1.0.6-feature-name.1
```

## Examples

Check out the [examples directory](examples/workflows/) for complete workflow examples:
- Simple version tagging
- Docker image builds
- Multi-stage deployments
- Monorepo versioning

## License

See [LICENSE](LICENSE) file in the main repository.

## Docker Images

The action uses pre-built Docker images for fast execution:

- **Action Image**: `ghcr.io/trondhindenes/autoversion:latest-action`
- **CLI Image**: `ghcr.io/trondhindenes/autoversion:latest`

Both images are automatically built and published on each release with versioned tags:
- `ghcr.io/trondhindenes/autoversion:1.0.0-action`
- `ghcr.io/trondhindenes/autoversion:1.0-action`
- `ghcr.io/trondhindenes/autoversion:1-action`
- `ghcr.io/trondhindenes/autoversion:latest-action`

### Version Pinning

You can control which version of autoversion to use:

```yaml
# Use latest major version (recommended - gets updates)
uses: trondhindenes/autoversion@v1
with:
  version: '1'  # Uses ghcr.io/trondhindenes/autoversion:1-action

# Use specific minor version
uses: trondhindenes/autoversion@v1
with:
  version: '1.0'  # Uses ghcr.io/trondhindenes/autoversion:1.0-action

# Use exact version (most stable)
uses: trondhindenes/autoversion@v1
with:
  version: '1.0.5'  # Uses ghcr.io/trondhindenes/autoversion:1.0.5-action

# Use latest (not recommended for production)
uses: trondhindenes/autoversion@v1
with:
  version: 'latest'  # Uses ghcr.io/trondhindenes/autoversion:latest-action
```

**Recommendation**: Use `version: '1'` (default) to automatically get patch updates while staying on the same major version.

## Support

For issues and questions:
- 📖 [Full Documentation](README.md)
- 🐛 [Report Issues](https://github.com/trondhindenes/autoversion/issues)
- 💬 [Discussions](https://github.com/trondhindenes/autoversion/discussions)
