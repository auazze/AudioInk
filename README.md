<p align="center">
  <img src="logo.svg" alt="AudioInk" width="160">
</p>

# AudioInk

Lightweight desktop app that fixes audio file metadata. Parses filenames like `Artist - Song (feat. Someone).mp3` and writes correct ID3/Vorbis tags — artist, title, track number — then saves a clean copy with a proper filename. Works as a GUI app or headless via `--fix` flag with right-click context menu integration on all platforms.

## Why

Audio files often have filenames like `01. Tomoya Ohtani-Break Through It All (feat. Kellin Quinn).mp3` but empty metadata tags. Players (iPhone, AIMP, Spotify local files) show blank fields. AudioInk fixes this automatically.

## What It Does

| Input filename | Output filename | Artist tag | Title tag |
|---|---|---|---|
| `Tomoya Ohtani-Break Through It All (feat. Kellin Quinn).mp3` | `Tomoya Ohtani & Kellin Quinn - Break Through It All.mp3` | Tomoya Ohtani & Kellin Quinn | Break Through It All |
| `01. Queen - Bohemian Rhapsody (Live).flac` | `Queen - Bohemian Rhapsody (Live).flac` | Queen | Bohemian Rhapsody (Live) |
| `DJ Snake feat. Lil Jon - Turn Down for What (Remix).mp3` | `DJ Snake & Lil Jon - Turn Down for What (Remix).mp3` | DJ Snake & Lil Jon | Turn Down for What (Remix) |
| `Кино - Группа крови.mp3` | `Кино - Группа Крови.mp3` | Кино | Группа Крови |
| `aleksandr_novikov_roza_713893675_456239280.mp3` | `Aleksandr Novikov - Roza.mp3` | Aleksandr Novikov | Roza |
| `~~~~.mp3` | Manual entry dialog | User-provided | User-provided |

### Parser handles

- **Separators**: ` - `, ` — `, ` – `, `_-_`, `+-+`, `--`, `==`, bare dashes
- **Featured artists**: `feat.`, `ft.`, `featuring` — merged with `&`
- **Track numbers**: `01.`, `01 -`, `#01`, `01_` — stripped
- **Extras**: `(Remix)`, `(Live)`, `(Acoustic)`, `[Explicit]` — kept in title
- **Junk extras**: bitrate tags (`320kbps`), format tags (`FLAC`), URLs, `official video` — stripped
- **Copy suffixes**: `— копия`, `- Copy`, `(2)` — stripped
- **Garbage IDs**: trailing numeric/hex IDs from VK/SoundCloud — stripped
- **Underscore filenames**: `artist_name_title.mp3` — underscores replaced, heuristic artist/title split
- **Title Case**: per-word normalization, preserves abbreviations (DJ, NF, MC) and mixed case (McDonald)
- **Confidence scoring**: high/medium/low — uncertain parses highlighted for review
- **Garbage filenames**: pure garbage (`~~~~.mp3`, `12345678.mp3`) triggers manual entry dialog

### Two save modes

- **Save copies** (default) — originals stay untouched, clean copies go to `AudioInk/` folder next to your files
- **Fix originals** — renames and tags original files in place

## Supported Formats

MP3, FLAC, OGG, M4A, WAV, WMA, OPUS

## Tech Stack

| Component | Choice |
|---|---|
| Language | Go |
| GUI | Wails v2 (Go backend + web frontend) |
| Frontend | Svelte |
| Audio tags | [go-taglib](https://pkg.go.dev/go.senan.xyz/taglib) (TagLib via Wasm, no CGo) |

## Build & Install

Requirements: Go 1.23+, Node.js, [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
git clone https://github.com/auazze/AudioInk.git
cd AudioInk

# Install Wails CLI (once)
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build executable only
wails build

# Dev mode with hot reload
wails dev

# Run tests
go test ./...
```

Binary output: `build/bin/AudioInk` (or `AudioInk.exe` on Windows)

### Windows installer

Requires [NSIS](https://nsis.sourceforge.io/Download) in PATH.

```bash
wails build --nsis
```

Creates `build/bin/AudioInk-amd64-installer.exe` — installs the app, adds right-click context menu for audio files ("AudioInk Fix"), handles reinstall/uninstall via registry.

### macOS installer

Build on a Mac, then run the install script:

```bash
wails build
bash build/darwin/install.sh
```

Copies `AudioInk.app` to `/Applications` and installs a Finder Quick Action (right-click menu). Detects existing installation, offers reinstall/uninstall.

### Linux installer

Build on Linux, then run the install script:

```bash
wails build
bash build/linux/install-context-menu.sh
```

Installs the binary and adds context menus for Nautilus (GNOME), Dolphin (KDE), and Freedesktop `.desktop` entry. Detects existing installation, offers reinstall/uninstall.

## Usage

### GUI

1. Launch AudioInk
2. Drag & drop audio files onto the window, or use **Select files** / **Select folder**
3. If any files have garbage names, a manual entry dialog pops up — enter Artist + Title or skip
4. Review parsed results in the table — double-click to edit artist/title
5. Click **Apply Tags** — choose **Save copies** or **Fix originals**
6. For copies: find clean files in `AudioInk/` next to your originals

### CLI

```bash
AudioInk --fix file1.mp3 file2.flac ...
```

Runs headless: parses filenames, writes tags, renames files in place. For garbage filenames, a native OS dialog (PowerShell WinForms / zenity / osascript) prompts for Artist + Title. Also used by right-click context menu integrations.
