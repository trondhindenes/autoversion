# autoversion

A Go-based CLI tool that automatically generates semantic versions based on the state of a git repository.

## Features

- Generates semantic versions (semver) based on git commit history
- Git tag support: tags on commits take precedence over calculated versions
- Configurable tag prefix stripping (e.g., `PRODUCT/2.0.0` → `2.0.0`)
- Main branch versions: `1.0.0`, `1.0.1`, `1.0.2`, etc.
- Feature branch prerelease versions: `1.0.2-feature.0`, `1.0.2-feature.1`, etc.
- Configurable main branch name (defaults to `main`)
- Supports both YAML and JSON configuration files
- JSON schema generation for configuration validation
- Makefile with build, test, and development targets

## Installation

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

Create a configuration file named `.autoversion.yaml` or `.autoversion.json` in your repository root:

**YAML Example:**
```yaml
mainBranch: main
tagPrefix: "v"  # Optional: strip "v" prefix from tags (e.g., v2.0.0 → 2.0.0)
```

**JSON Example:**
```json
{
  "mainBranch": "main",
  "tagPrefix": "PRODUCT/"
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
- The tag name is used directly as the version
- If `tagPrefix` is configured, it's stripped from the tag name
- Example: With `tagPrefix: "v"`, tag `v2.0.0` becomes version `2.0.0`
- Example: With `tagPrefix: "PRODUCT/"`, tag `PRODUCT/3.1.0` becomes version `3.1.0`
- Tags take precedence regardless of branch

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

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `mainBranch` | string | `main` | The name of the main/primary branch |
| `tagPrefix` | string | `""` | Prefix to strip from git tags (e.g., `"v"` or `"PRODUCT/"`) |

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
$ git tag -a v2.0.0 -m "Release 2.0.0"
$ autoversion
v2.0.0

# With tagPrefix configured as "v"
$ autoversion
2.0.0
```

### With tag prefix (e.g., PRODUCT/):
```yaml
# .autoversion.yaml
mainBranch: main
tagPrefix: "PRODUCT/"
```

```bash
$ git tag -a PRODUCT/3.1.0 -m "Product Release 3.1.0"
$ autoversion
3.1.0
```

## Development

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
go test -v

# Build manually
go build -o bin/autoversion .
```

## License

MIT
