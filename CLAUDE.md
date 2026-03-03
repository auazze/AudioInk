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
main.go          — Wails app entry point, window config, drag & drop
app.go           — Backend API exposed to frontend (scan, parse, apply)
parser/          — Filename parser: separators, track numbers, extras, featured artists, confidence
tagger/          — Read/write audio metadata via go-taglib
scanner/         — Recursive directory scanner, extension filter
frontend/src/    — Svelte UI: DropZone, FileTable, EditRow
```

## Key Patterns

- **Safe output**: Files are copied to `_AudioInk_output/` folder, originals untouched
- **Featured artists**: `feat.`/`ft.` merged with main artist via `&`
- **Extras**: `(Remix)`, `(Live)` etc. appended to end of filename and title tag
- **Copy suffixes**: `— копия`, `- Copy`, `(2)` stripped automatically
- **Drag & drop**: Uses Wails native `OnFileDrop` with `DisableWebViewDrop: true`
- **Logging**: `audioink.log` created next to the exe

## Commands

```bash
# Dev mode
wails dev

# Build
wails build

# Run tests
go test ./...

# Run specific package tests
go test ./parser/ -v
```

## File Naming Convention

Output format: `Artist1 & Artist2 - Title (Extra).ext`
- All featured artists joined with ` & `
- Track numbers stripped
- Copy suffixes stripped
- Extras (Remix, Live, Acoustic, Remastered) in parentheses at end
