#!/bin/bash
# AudioInk installer for macOS
# Installs AudioInk.app to /Applications/ and the Quick Action to ~/Library/Services/
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_NAME="AudioInk"
APP_PATH="/Applications/${APP_NAME}.app"
WORKFLOW_NAME="AudioInk Fix"
WORKFLOW_SRC="$SCRIPT_DIR/${WORKFLOW_NAME}.workflow"
WORKFLOW_DST="$HOME/Library/Services/${WORKFLOW_NAME}.workflow"

already_installed=false

# Check if AudioInk is already installed
if [ -d "$APP_PATH" ] || [ -d "$WORKFLOW_DST" ]; then
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
            [ -d "$APP_PATH" ] && rm -rf "$APP_PATH"
            [ -d "$WORKFLOW_DST" ] && rm -rf "$WORKFLOW_DST"
            echo "Old installation removed."
            ;;
        2)
            echo "Uninstalling AudioInk..."
            [ -d "$APP_PATH" ] && rm -rf "$APP_PATH"
            [ -d "$WORKFLOW_DST" ] && rm -rf "$WORKFLOW_DST"
            echo "AudioInk has been uninstalled."
            exit 0
            ;;
        *)
            echo "Cancelled."
            exit 0
            ;;
    esac
fi

# Install AudioInk.app
if [ -d "$SCRIPT_DIR/${APP_NAME}.app" ]; then
    echo "Installing ${APP_NAME}.app to /Applications/..."
    cp -R "$SCRIPT_DIR/${APP_NAME}.app" "$APP_PATH"
    echo "App installed."
else
    echo "Warning: ${APP_NAME}.app not found in $SCRIPT_DIR — skipping app install."
    echo "Copy ${APP_NAME}.app to /Applications/ manually."
fi

# Install Quick Action workflow
if [ -d "$WORKFLOW_SRC" ]; then
    mkdir -p "$HOME/Library/Services"
    cp -R "$WORKFLOW_SRC" "$WORKFLOW_DST"
    echo "Quick Action installed — right-click audio files in Finder to fix them."
else
    echo "Warning: ${WORKFLOW_NAME}.workflow not found — skipping Quick Action install."
fi

echo ""
echo "AudioInk installed successfully."
