#!/bin/bash
# Install AudioInk context menu entries on Linux
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Install .desktop file
mkdir -p ~/.local/share/applications
cp "$SCRIPT_DIR/audioink-fix.desktop" ~/.local/share/applications/
update-desktop-database ~/.local/share/applications/ 2>/dev/null || true

# Install Nautilus script (if Nautilus is the file manager)
if command -v nautilus &>/dev/null; then
    mkdir -p ~/.local/share/nautilus/scripts
    cp "$SCRIPT_DIR/nautilus-audioink-fix.sh" "$HOME/.local/share/nautilus/scripts/AudioInk Fix"
    chmod +x "$HOME/.local/share/nautilus/scripts/AudioInk Fix"
    echo "Nautilus script installed."
fi

# Install Dolphin service menu (if Dolphin/KDE is the file manager)
if command -v dolphin &>/dev/null; then
    mkdir -p ~/.local/share/kio/servicemenus
    cat > ~/.local/share/kio/servicemenus/audioink-fix.desktop << 'INNER_EOF'
[Desktop Entry]
Type=Service
MimeType=audio/mpeg;audio/flac;audio/ogg;audio/mp4;audio/x-wav;audio/x-ms-wma;audio/opus;
Actions=fixAudioInk

[Desktop Action fixAudioInk]
Name=AudioInk: Fix name & tags
Exec=audioink --fix %F
INNER_EOF
    echo "Dolphin service menu installed."
fi

echo "AudioInk context menu installed successfully."
