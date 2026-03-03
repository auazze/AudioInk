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
main.go          — Entry point: --fix flag → CLI mode, otherwise GUI
app.go           — Backend API exposed to frontend (scan, parse, apply)
fix.go           — Headless CLI fix logic (parse, tag, rename)
notify.go        — Cross-platform system notifications (Windows/macOS/Linux)
parser/          — Filename parser: separators, track numbers, extras, featured artists, confidence
tagger/          — Read/write audio metadata via go-taglib
scanner/         — Recursive directory scanner, extension filter
frontend/src/    — Svelte UI: DropZone, FileTable, EditRow
build/windows/   — NSIS installer + context menu registry entries
build/darwin/    — macOS Quick Action for Finder context menu
build/linux/     — Freedesktop .desktop + Nautilus/Dolphin scripts
```

## Key Patterns

- **Two save modes**: Copy to `AudioInk/` subfolder (originals untouched) or fix originals in place
- **Featured artists**: `feat.`/`ft.` merged with main artist via `&`
- **Extras**: `(Remix)`, `(Live)` etc. appended to end of filename and title tag
- **Copy suffixes**: `— копия`, `- Copy`, `(2)` stripped automatically
- **Drag & drop**: Uses Wails native `OnFileDrop` with `DisableWebViewDrop: true`
- **Logging**: `audioink.log` created next to the exe

## Commands

```bash
# Dev mode (GUI)
wails dev

# Build
wails build

# Run tests
go test ./...

# Run specific package tests
go test ./parser/ -v

# CLI mode: fix files without GUI
AudioInk --fix file1.mp3 file2.mp3
```

## CLI Mode (--fix)

`AudioInk --fix file1.mp3 file2.flac ...` runs headless: parses filenames, writes tags, renames files in place, shows a system toast notification with the result. Used by OS context menu integrations.

Without arguments, AudioInk opens the GUI as usual.

### Context Menu Files

- **Windows**: `build/windows/installer/project.nsi` — registry entries via NSIS
- **macOS**: `build/darwin/AudioInk Fix.workflow/` — Automator Quick Action
- **Linux**: `build/linux/` — Freedesktop .desktop + Nautilus/Dolphin scripts

## File Naming Convention

Output format: `Artist1 & Artist2 - Title (Extra).ext`
- All featured artists joined with ` & `
- Track numbers stripped
- Copy suffixes stripped
- Extras (Remix, Live, Acoustic, Remastered) in parentheses at end
