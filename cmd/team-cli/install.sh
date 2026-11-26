#!/bin/bash
# Team CLI Installation Script

set -e

REPO_URL="https://github.com/thearchitectit/agent-guardrails-template"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../.." && pwd)"
if [[ ${PREFIX+x} && "$PREFIX" =~ ^[[:space:]]*$ ]]; then
    echo "PREFIX must not be empty or whitespace" >&2
    exit 2
fi
PREFIX="${PREFIX-/usr/local}"
BINDIR="$PREFIX/bin"
DATADIR="$PREFIX/share/agent-guardrails-template/team-cli"
BINARY_NAME="team"
BINARY_NAME_ALT="team-cli"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        PLATFORM="linux"
        ;;
    darwin)
        PLATFORM="darwin"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Installing team CLI for $PLATFORM-$ARCH..."

# Validate the complete backend before creating directories or installing a binary.
for backend_file in team_manager.py batch_operations.py encryption.py migrations/__init__.py migrations/v1_to_v2.py; do
    if [ ! -f "$REPO_ROOT/scripts/$backend_file" ]; then
        echo "Python backend payload is incomplete: $REPO_ROOT/scripts/$backend_file" >&2
        exit 1
    fi
done

# Check if running from source
if [ -f "$REPO_ROOT/cmd/team-cli/team" ]; then
    echo "Installing from local build..."
    BINARY_SOURCE="$REPO_ROOT/cmd/team-cli/team"
elif [ -f "team" ]; then
    echo "Installing from current directory..."
    BINARY_SOURCE="$(pwd)/team"
else
    echo "Binary not found. Please build from source with:"
    echo "  cd cmd/team-cli && make build"
    exit 1
fi

mkdir -p "$BINDIR" "$DATADIR/scripts/migrations"
install -m 644 "$REPO_ROOT/scripts/team_manager.py" "$DATADIR/scripts/team_manager.py"
install -m 644 "$REPO_ROOT/scripts/batch_operations.py" "$DATADIR/scripts/batch_operations.py"
install -m 644 "$REPO_ROOT/scripts/encryption.py" "$DATADIR/scripts/encryption.py"
install -m 644 "$REPO_ROOT/scripts/migrations/__init__.py" "$DATADIR/scripts/migrations/__init__.py"
install -m 644 "$REPO_ROOT/scripts/migrations/v1_to_v2.py" "$DATADIR/scripts/migrations/v1_to_v2.py"
install -m 755 "$BINARY_SOURCE" "$BINDIR/$BINARY_NAME"
ln -sf "$BINARY_NAME" "$BINDIR/$BINARY_NAME_ALT"

echo "team CLI and Python backend installed under $PREFIX"
echo ""
echo "Run 'team --help' to get started"
