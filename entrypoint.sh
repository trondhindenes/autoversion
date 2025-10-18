#!/bin/bash
set -e

# Input parameters
CONFIG_FILE="$1"
FAIL_ON_ERROR="${2:-true}"

# Build autoversion command
CMD="autoversion"
if [ -n "$CONFIG_FILE" ]; then
  CMD="$CMD --config $CONFIG_FILE"
fi

# Run autoversion and capture both stdout (version) and stderr (logs)
TEMP_OUTPUT=$(mktemp)
TEMP_ERROR=$(mktemp)

if $CMD > "$TEMP_OUTPUT" 2> "$TEMP_ERROR"; then
  VERSION=$(cat "$TEMP_OUTPUT")

  # Show the logs from stderr
  cat "$TEMP_ERROR" >&2

  echo "Calculated version: $VERSION"

  # Parse version components
  # Format can be: 1.0.0 or 1.0.0-pre.0 or 1.0.0-feature.1

  # Extract base version (before any -)
  BASE_VERSION="${VERSION%%-*}"

  # Split major.minor.patch
  IFS='.' read -r MAJOR MINOR PATCH <<< "$BASE_VERSION"

  # Check if there's a prerelease part
  if [[ "$VERSION" == *"-"* ]]; then
    PRERELEASE="${VERSION#*-}"
    IS_PRERELEASE="true"
  else
    PRERELEASE=""
    IS_PRERELEASE="false"
  fi

  # Set GitHub Action outputs
  echo "version=$VERSION" >> "$GITHUB_OUTPUT"
  echo "major=$MAJOR" >> "$GITHUB_OUTPUT"
  echo "minor=$MINOR" >> "$GITHUB_OUTPUT"
  echo "patch=$PATCH" >> "$GITHUB_OUTPUT"
  echo "prerelease=$PRERELEASE" >> "$GITHUB_OUTPUT"
  echo "is-prerelease=$IS_PRERELEASE" >> "$GITHUB_OUTPUT"

  # Clean up
  rm -f "$TEMP_OUTPUT" "$TEMP_ERROR"

  exit 0
else
  # Show error output
  cat "$TEMP_ERROR" >&2

  # Clean up
  rm -f "$TEMP_OUTPUT" "$TEMP_ERROR"

  if [ "$FAIL_ON_ERROR" = "true" ]; then
    echo "Error: Failed to calculate version" >&2
    exit 1
  else
    echo "Warning: Failed to calculate version, but continuing due to fail-on-error=false" >&2

    # Set empty outputs
    echo "version=" >> "$GITHUB_OUTPUT"
    echo "major=" >> "$GITHUB_OUTPUT"
    echo "minor=" >> "$GITHUB_OUTPUT"
    echo "patch=" >> "$GITHUB_OUTPUT"
    echo "prerelease=" >> "$GITHUB_OUTPUT"
    echo "is-prerelease=false" >> "$GITHUB_OUTPUT"

    exit 0
  fi
fi
