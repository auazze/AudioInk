#!/bin/bash
# AudioInk installer for Linux
# Installs the binary and context menu entries for Nautilus/Dolphin
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BIN_NAME="audioink"
BIN_PATH="/usr/local/bin/$BIN_NAME"
DESKTOP_FILE="$HOME/.local/share/applications/audioink-fix.desktop"
NAUTILUS_SCRIPT="$HOME/.local/share/nautilus/scripts/AudioInk Fix"
DOLPHIN_SERVICE="$HOME/.local/share/kio/servicemenus/audioink-fix.desktop"

already_installed=false

# Check if AudioInk is already installed
if [ -f "$BIN_PATH" ] || [ -f "$DESKTOP_FILE" ] || [ -f "$NAUTILUS_SCRIPT" ] || [ -f "$DOLPHIN_SERVICE" ]; then
    already_installed=true
fi

if $already_installed; then
    echo ""
    echo "AudioInk is already installed."
    echo ""
    echo "  1) Reinstall (remove old, install new)"
    echo "  2) Uninstall only"
    echo "  3) Cancel"
    echo ""
    printf "Choose [1/2/3]: "
    read -r choice

    case "$choice" in
        1)
            echo "Removing old installation..."
            [ -f "$BIN_PATH" ] && sudo rm -f "$BIN_PATH"
            [ -f "$DESKTOP_FILE" ] && rm -f "$DESKTOP_FILE"
            [ -f "$NAUTILUS_SCRIPT" ] && rm -f "$NAUTILUS_SCRIPT"
            [ -f "$DOLPHIN_SERVICE" ] && rm -f "$DOLPHIN_SERVICE"
            update-desktop-database ~/.local/share/applications/ 2>/dev/null || true
            echo "Old installation removed."
            ;;
        2)
            echo "Uninstalling AudioInk..."
            [ -f "$BIN_PATH" ] && sudo rm -f "$BIN_PATH"
            [ -f "$DESKTOP_FILE" ] && rm -f "$DESKTOP_FILE"
            [ -f "$NAUTILUS_SCRIPT" ] && rm -f "$NAUTILUS_SCRIPT"
            [ -f "$DOLPHIN_SERVICE" ] && rm -f "$DOLPHIN_SERVICE"
            update-desktop-database ~/.local/share/applications/ 2>/dev/null || true
            echo "AudioInk has been uninstalled."
            exit 0
            ;;
        *)
            echo "Cancelled."
            exit 0
            ;;
    esac
fi

# Install binary
if [ -f "$SCRIPT_DIR/$BIN_NAME" ]; then
    echo "Installing $BIN_NAME to /usr/local/bin/ (requires sudo)..."
    sudo cp "$SCRIPT_DIR/$BIN_NAME" "$BIN_PATH"
    sudo chmod +x "$BIN_PATH"
    echo "Binary installed."
else
    echo "Warning: $BIN_NAME binary not found in $SCRIPT_DIR — skipping binary install."
    echo "Make sure 'audioink' is in your PATH."
fi

# Install .desktop file
mkdir -p ~/.local/share/applications
cp "$SCRIPT_DIR/audioink-fix.desktop" "$DESKTOP_FILE"
update-desktop-database ~/.local/share/applications/ 2>/dev/null || true

# Install Nautilus script (if Nautilus is the file manager)
if command -v nautilus &>/dev/null; then
    mkdir -p ~/.local/share/nautilus/scripts
    cp "$SCRIPT_DIR/nautilus-audioink-fix.sh" "$NAUTILUS_SCRIPT"
    chmod +x "$NAUTILUS_SCRIPT"
    echo "Nautilus context menu installed."
fi

# Install Dolphin service menu (if Dolphin/KDE is the file manager)
if command -v dolphin &>/dev/null; then
    mkdir -p ~/.local/share/kio/servicemenus
    cat > "$DOLPHIN_SERVICE" << 'INNER_EOF'
[Desktop Entry]
Type=Service
MimeType=audio/mpeg;audio/flac;audio/ogg;audio/mp4;audio/x-wav;audio/x-ms-wma;audio/opus;
Actions=fixAudioInk

[Desktop Action fixAudioInk]
Name=AudioInk: Fix name & tags
Exec=audioink --fix %F
INNER_EOF
    echo "Dolphin context menu installed."
fi

echo ""
echo "AudioInk installed successfully."
