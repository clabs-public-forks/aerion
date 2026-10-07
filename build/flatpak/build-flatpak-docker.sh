#!/bin/bash
# Build Aerion Flatpak using Docker (hybrid approach: build binary on host in container, then package)

set -e

cd "$(dirname "$0")/../.."

echo "=== Aerion Flatpak Docker Builder ==="
echo ""

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed"
    echo ""
    echo "Install it with:"
    echo "  Ubuntu/Debian: sudo apt install docker.io"
    echo "  Fedora:        sudo dnf install docker"
    echo "  Arch:          sudo pacman -S docker"
    exit 1
fi

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo "❌ Docker daemon is not running"
    echo ""
    echo "Start it with:"
    echo "  sudo systemctl start docker"
    exit 1
fi

# Check for OAuth credentials
if [ -z "$GOOGLE_CLIENT_ID" ] && [ -z "$MICROSOFT_CLIENT_ID" ]; then
    echo "⚠️  Warning: No OAuth credentials found"
    echo "Gmail and Outlook OAuth will not work in the built app"
    echo ""
fi

echo "Building Docker image (this may take a few minutes on first run)..."
docker build -t aerion-flatpak-builder -f build/flatpak/Dockerfile build/flatpak

echo ""
echo "Building Aerion in Docker container..."
echo ""

# Get version from git tag
VERSION=$(git describe --tags --exact-match 2>/dev/null || echo "dev")

# Run the build in Docker
docker run --rm --privileged \
    -v "$(pwd):/workspace" \
    -w /workspace \
    -e GOOGLE_CLIENT_ID="${GOOGLE_CLIENT_ID}" \
    -e GOOGLE_CLIENT_SECRET="${GOOGLE_CLIENT_SECRET}" \
    -e MICROSOFT_CLIENT_ID="${MICROSOFT_CLIENT_ID}" \
    -e GOOGLE_TESTING_CLIENT_ID="${GOOGLE_TESTING_CLIENT_ID}" \
    -e GOOGLE_TESTING_CLIENT_SECRET="${GOOGLE_TESTING_CLIENT_SECRET}" \
    -e HOST_UID="$(id -u)" \
    -e HOST_GID="$(id -g)" \
    aerion-flatpak-builder \
    bash -c "
        set -e
        # The container runs as root; on exit, remove the generated credentials
        # file and return build outputs in the bind mount to the host user.
        cleanup() {
            rm -f internal/oauth2/credentials_gen.go
            chown -R \"\$HOST_UID:\$HOST_GID\" frontend/node_modules frontend/dist frontend/wailsjs build/bin repo build-dir .flatpak-builder 2>/dev/null || true
        }
        trap cleanup EXIT

        echo 'Installing frontend dependencies...'
        (cd frontend && npm ci)

        echo ''
        echo 'Building Aerion binary...'
        make build-linux

        echo ''
        echo 'Packaging into Flatpak...'
        # The dev manifest packages the binary built above; the flathub
        # manifest would rebuild a tagged upstream release from git instead.
        flatpak-builder --force-clean --disable-rofiles-fuse --repo=repo build-dir build/flatpak/io.github.hkdb.Aerion-dev.yml

        echo ''
        echo 'Creating .flatpak bundle...'
        mkdir -p build/bin
        flatpak build-bundle repo build/bin/Aerion-${VERSION}.flatpak io.github.hkdb.Aerion
    "

echo ""
echo "✅ Build complete!"
echo ""
echo "Flatpak bundle created: build/bin/Aerion-${VERSION}.flatpak"
echo ""
echo "To install locally:"
echo "  flatpak install --user build/bin/Aerion-${VERSION}.flatpak"
echo ""
echo "To run:"
echo "  flatpak run io.github.hkdb.Aerion"
