# AudioInk

## Project Overview

Desktop app that parses audio filenames and writes correct metadata tags. Built with Go + Wails v2 (Svelte frontend).

**Input**: `Tomoya Ohtani-Break Through It All (feat. Kellin Quinn).mp3`
**Output**: `Tomoya Ohtani & Kellin Quinn - Break Through It All.mp3` with correct artist/title tags.

## Tech Stack

- **Backend**: Go 1.23+, Wails v2
- **Frontend**: Svelte 3, Vite
- **Audio tags**: go.senan.xyz/taglib (TagLib via Wasm, no CGo)

## Project Structure

```
main.go              — Entry point: --fix flag → CLI mode, otherwise GUI
app.go               — Backend API exposed to frontend (scan, parse, apply, ApplyQuick, confirm mode methods)
confirm.go           — showConfirmDialog(): launches Wails window for CLI confirm (cross-platform)
fix.go               — Headless CLI fix logic: batch parse → batch confirm → batch apply
dialog.go            — ManualEntry struct
notify.go            — Notification stub (currently no-op)
parser/              — Filename parser: separators, track numbers, extras, featured artists, confidence
tagger/              — Read/write audio metadata via go-taglib
scanner/             — Recursive directory scanner, extension filter
frontend/src/        — Svelte UI: DropZone, FileTable, EditRow, ManualEntryDialog, ConfirmView
build/windows/       — NSIS installer: context menu, auto-kill processes, reinstall/uninstall
build/darwin/        — macOS: install.sh (app + Quick Action), existing install detection
build/linux/         — Linux: install-context-menu.sh (binary + Nautilus/Dolphin), existing install detection
```

## Key Patterns

- **Two save modes**: Copy to `AudioInk/` subfolder (originals untouched) or fix originals in place
- **Confirm dialog (CLI)**: Low-confidence files open a Wails confirm window (ConfirmView.svelte) with prefilled artist/title — same dark theme as main app
- **Confirm dialog (GUI)**: ManualEntryDialog.svelte shown as overlay for low-confidence files with prefilled suggestions
- **App.confirmMode**: When true, frontend shows ConfirmView instead of main app UI
- **Batch confirm**: fix.go parses all files first, collects low-confidence ones, shows ONE confirm dialog for all, then applies results
- **ApplyQuick**: Single-file overwrite method for GUI manual entry, delegates to processApply
- **Featured artists**: `feat.`/`ft.` merged with main artist via `&`
- **Extras**: `(Remix)`, `(Live)` etc. appended to end of filename and title tag
- **Junk extras stripped**: Bitrate tags, format tags, URLs, "official video" etc.
- **Drag & drop**: Uses Wails native `OnFileDrop` with `DisableWebViewDrop: true`
- **Case-safe rename**: `pathsEqual()` for case-insensitive path comparison on Windows (avoids spurious `(2)` suffix)
- **Logging**: `audioink.log` next to the exe, append mode, shared by GUI and CLI

## Commands

```bash
# Dev mode (GUI)
wails dev

# Build (exe only)
wails build

# Build with Windows installer (requires NSIS — add to PATH if needed)
export PATH="$PATH:/c/Program Files (x86)/NSIS/Bin:/c/Program Files (x86)/NSIS"
wails build --nsis

# Run tests
go test ./...

# Run specific package tests
go test ./parser/ -v

# CLI mode: fix files without GUI
AudioInk --fix file1.mp3 file2.mp3
```

## CLI Mode (--fix)

`AudioInk --fix file1.mp3 file2.flac ...` runs headless: parses filenames, shows confirm dialog for uncertain files, writes tags, renames files in place. Used by OS context menu integrations.

Without arguments, AudioInk opens the GUI as usual.

### Installers (all detect existing installations, offer reinstall/uninstall)

- **Windows**: `build/windows/installer/project.nsi` — NSIS installer with registry-based detection, auto-kills running processes
- **macOS**: `build/darwin/install.sh` — copies .app + Quick Action, checks /Applications/
- **Linux**: `build/linux/install-context-menu.sh` — installs binary + context menus, checks paths

### Context Menu Files

- **Windows**: Registry entries via NSIS (`HKCR\SystemFileAssociations`), icon.ico for fast menu rendering
- **macOS**: `build/darwin/AudioInk Fix.workflow/` — Automator Quick Action for Finder
- **Linux**: Nautilus scripts + Dolphin service menus + Freedesktop .desktop

## Parser Features

- Per-word Title Case (ALL CAPS/lowercase words → Title Case, abbreviations DJ/NF/MC preserved, mixed case untouched)
- Trailing suffix stripping (`_01` track suffixes, `-21498` garbage IDs, hex IDs)
- Underscore filenames: `artist_name_title.mp3` → heuristic artist/title split (Confidence=Low)
- Hyphenated filenames: `artist-song-title.mp3` → first segment as artist (Confidence=Low)
- Compound separator normalization (`--`, `==`, `+-+` → single separator)
- Junk extras stripping (bitrate, format tags, URLs, "official video")
- Hyphenated name cleanup (`police-siren` → `Police Siren`)
- Copy suffix stripping (`— копия`, `- Copy`, `(2)`)
- Featured artist extraction (`feat.`/`ft.` in artist or title)
- `+` in artist → `&` (e.g. `Artist1 + Artist2` → `Artist1 & Artist2`)

## File Naming Convention

Output format: `Artist1 & Artist2 - Title (Extra).ext`
- All featured artists joined with ` & `
- Track numbers stripped
- Copy suffixes stripped
- Extras (Remix, Live, Acoustic, Remastered) in parentheses at end
