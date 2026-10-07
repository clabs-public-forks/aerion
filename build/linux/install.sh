#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

echo -e "${PURPLE}
 ░▒▓██████▓▒░░▒▓████████▓▒░▒▓███████▓▒░░▒▓█▓▒░░▒▓██████▓▒░░▒▓███████▓▒░  
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓████████▓▒░▒▓██████▓▒░ ░▒▓███████▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░      ░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓█▓▒░ 
░▒▓█▓▒░░▒▓█▓▒░▒▓████████▓▒░▒▓█▓▒░░▒▓█▓▒░▒▓█▓▒░░▒▓██████▓▒░░▒▓█▓▒░░▒▓█▓▒░ 
                                                                         
${NC}
"

# Print colored output
print_error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}$1${NC}"
}

print_info() {
    echo -e "${YELLOW}$1${NC}"
}

# Check if running on Linux
if [[ "$(uname -s)" != "Linux" ]]; then
    print_error "This script is only for Linux systems"
    exit 1
fi

# Check if binary exists
if [[ ! -f "aerion" ]]; then
    print_error "aerion binary not found in current directory"
    echo "Please run this script from the directory containing the aerion binary"
    exit 1
fi

# Check if desktop file exists
if [[ ! -f "io.github.clabs_public_forks.Aerion.desktop" ]]; then
    print_error "io.github.clabs_public_forks.Aerion.desktop file not found in current directory"
    echo "Please ensure the desktop file is in the same directory as this script"
    exit 1
fi

# Check if icons exist. New tarballs ship sized icons under icons/ plus a
# scalable SVG (#395); the root 256px PNG remains for backward compatibility.
if [[ ! -f "io.github.clabs_public_forks.Aerion.png" ]]; then
    print_error "io.github.clabs_public_forks.Aerion.png icon not found in current directory"
    echo "Please ensure the icon file is in the same directory as this script"
    exit 1
fi

# hicolor icon sizes (#395). Keep in sync with ICON_SIZES in the Makefile and
# .github/workflows/release.yml.
ICON_SIZES="32 48 64 128 256"

echo ""
print_info "Aerion Email Client - Installation Script"
echo ""
echo "This script will install Aerion on your system."
echo ""
echo "Choose installation type:"
echo "  1) System-wide (requires sudo, installs to /usr/local)"
echo "  2) User only (installs to ~/.local)"
echo ""

while true; do
    read -p "Enter your choice (1 or 2): " choice
    case $choice in
        1)
            INSTALL_TYPE="system"
            BIN_DIR="/usr/local/bin"
            APPS_DIR="/usr/share/applications"
            ICONS_BASE="/usr/share/icons/hicolor"
            NEEDS_SUDO=true
            break
            ;;
        2)
            INSTALL_TYPE="user"
            BIN_DIR="$HOME/.local/bin"
            APPS_DIR="$HOME/.local/share/applications"
            ICONS_BASE="$HOME/.local/share/icons/hicolor"
            NEEDS_SUDO=false
            break
            ;;
        *)
            print_error "Invalid choice. Please enter 1 or 2."
            ;;
    esac
done

echo ""
print_info "Installing Aerion ($INSTALL_TYPE)..."
echo ""

# Function to run command with or without sudo
run_cmd() {
    if [[ "$NEEDS_SUDO" == true ]]; then
        sudo "$@"
    else
        "$@"
    fi
}

# Check for old desktop file and rename it (backwards compatibility)
OLD_DESKTOP_FILE="$APPS_DIR/aerion.desktop"
if [[ -f "$OLD_DESKTOP_FILE" ]]; then
    print_info "Found old aerion.desktop, renaming to aerion.desktop.backup..."
    run_cmd mv "$OLD_DESKTOP_FILE" "$APPS_DIR/aerion.desktop.backup"
    print_success "Old desktop file renamed to aerion.desktop.backup"
fi

# Remove files installed under the upstream app ID (io.github.hkdb.Aerion)
# so the menu doesn't show two Aerion entries
LEGACY_ID="io.github.hkdb.Aerion"
if [[ -f "$APPS_DIR/$LEGACY_ID.desktop" ]]; then
    print_info "Removing files installed under the old app ID $LEGACY_ID..."
    run_cmd rm -f "$APPS_DIR/$LEGACY_ID.desktop"
    for sz in $ICON_SIZES; do
        run_cmd rm -f "$ICONS_BASE/${sz}x${sz}/apps/$LEGACY_ID.png"
    done
    run_cmd rm -f "$ICONS_BASE/scalable/apps/$LEGACY_ID.svg"
fi

# Create directories if they don't exist (icon dirs are created per-size by
# install -D below)
print_info "Creating directories..."
run_cmd mkdir -p "$BIN_DIR"
run_cmd mkdir -p "$APPS_DIR"

# Install binary
print_info "Installing binary to $BIN_DIR..."
run_cmd install -Dm755 aerion "$BIN_DIR/aerion"

# Install desktop file
print_info "Installing desktop file to $APPS_DIR..."
run_cmd install -Dm644 io.github.clabs_public_forks.Aerion.desktop "$APPS_DIR/io.github.clabs_public_forks.Aerion.desktop"

# Install icons: all hicolor sizes when the tarball ships them (#395),
# falling back to the root 256px PNG for older tarballs
print_info "Installing icons to $ICONS_BASE..."
if [[ -d "icons" ]]; then
    for sz in $ICON_SIZES; do
        if [[ -f "icons/${sz}x${sz}/io.github.clabs_public_forks.Aerion.png" ]]; then
            run_cmd install -Dm644 "icons/${sz}x${sz}/io.github.clabs_public_forks.Aerion.png" \
                "$ICONS_BASE/${sz}x${sz}/apps/io.github.clabs_public_forks.Aerion.png"
        fi
    done
    if [[ -f "icons/scalable/io.github.clabs_public_forks.Aerion.svg" ]]; then
        run_cmd install -Dm644 "icons/scalable/io.github.clabs_public_forks.Aerion.svg" \
            "$ICONS_BASE/scalable/apps/io.github.clabs_public_forks.Aerion.svg"
    fi
else
    print_info "No sized icons in this package - installing 256px icon only"
    run_cmd install -Dm644 io.github.clabs_public_forks.Aerion.png "$ICONS_BASE/256x256/apps/io.github.clabs_public_forks.Aerion.png"
fi

# Update icon cache
print_info "Updating icon cache..."
if [[ "$INSTALL_TYPE" == "system" ]]; then
    run_cmd gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || true
else
    gtk-update-icon-cache -f -t "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
fi

# Update desktop database
print_info "Updating desktop database..."
if [[ "$INSTALL_TYPE" == "system" ]]; then
    run_cmd update-desktop-database /usr/share/applications 2>/dev/null || true
else
    update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
fi

echo ""
print_success "✓ Installation complete!"
echo ""

# Additional setup instructions
if [[ "$INSTALL_TYPE" == "user" ]]; then
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        print_info "Note: $HOME/.local/bin is not in your PATH"
        echo "You may need to add it to your PATH by adding this line to your ~/.bashrc or ~/.zshrc:"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo ""
    fi
fi

echo "You may need to log out and back in for the application to appear in your menu."
echo ""
echo "To set Aerion as your default email client, run:"
echo "  xdg-mime default io.github.clabs_public_forks.Aerion.desktop x-scheme-handler/mailto"
echo ""
echo "To start Aerion, run:"
echo "  aerion --dbus-notify"
echo ""
