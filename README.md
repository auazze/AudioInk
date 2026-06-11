<p align="center">
  <img src="logo.svg" alt="AudioInk" width="160">
</p>

# AudioInk

[![CI](https://github.com/auazze/AudioInk/actions/workflows/ci.yml/badge.svg)](https://github.com/auazze/AudioInk/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Lightweight desktop app that fixes audio file metadata. Parses filenames like `Artist - Song (feat. Someone).mp3` and writes correct ID3/Vorbis tags — artist, title, track number — then saves a clean copy with a proper filename. Works as a GUI app or headless via `--fix` flag with right-click context menu integration. Bundled FFmpeg adds a preview player, health checks, format conversion, silence trimming, repair, loudness normalization, and duplicate detection.

**Platform support:** Windows is the primary, tested platform with a [prebuilt installer](https://github.com/auazze/AudioInk/releases/latest). macOS and Linux build scripts and context-menu integrations exist (`build/darwin/`, `build/linux/`) but are **untested** — I have no Apple/Linux hardware to verify them. Bug reports and PRs welcome.

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
| `~~~~.mp3` | Flagged for review — you enter Artist + Title | User-provided | User-provided |

### Parser handles

- **Separators**: ` - `, ` — `, ` – `, `_-_`, `+-+`, `--`, `==`, ` | `, ` • `, bare dashes; em/en dash and full-width CJK punctuation normalized
- **Featured artists**: `feat.`, `ft.`, `featuring`, `(with X)` — merged with `&`, deduplicated
- **Track numbers**: `01.`, `01 -`, `#01`, `01_`, `Disc 1 Track 03` — stripped
- **Extras**: `(Remix)`, `(Live)`, `(Acoustic)`, `[Explicit]`, `(Slowed + Reverb)` — kept in title
- **Junk extras**: bitrate/format tags (`320kbps`, `FLAC`), URLs, YouTube noise (`Official Video`, `Lyrics`, `Visualizer`, `MV`, `1080p`, `NCS Release`, `Sub Español`) — stripped, bracketed or not
- **yt-dlp suffixes**: `Title [dQw4w9WgXcQ].mp3` — video-id stripped (real extras like `[Instrumental]` survive)
- **Download-site prefixes**: `y2mate.com - `, `[Mp3Juices.cc]`, `www.site.net_` — stripped (artists like `will.i.am` survive)
- **Quoted titles**: `Кино «Группа крови».mp3`, `米津玄師「Lemon」.mp3` — quotes treated as artist/title structure
- **Copy suffixes**: `— копия`, `- Copy`, `(2)` — stripped
- **Garbage IDs**: trailing numeric/hex IDs from VK/SoundCloud — stripped (years like `1999` survive)
- **Underscore filenames**: `artist_name_title.mp3` — underscores replaced, heuristic artist/title split
- **Title Case**: per-word normalization, preserves abbreviations (DJ, NF, MC), stylized names (P!nk, A$AP, will.i.am), Roman numerals, mixed case (McDonald)
- **Confidence scoring**: high/medium/low — uncertain parses highlighted for review
- **Garbage filenames**: pure garbage (`~~~~.mp3`, `12345678.mp3`) is flagged for review — inline edit in the GUI, confirm window in CLI mode

### Audio toolbox (bundled FFmpeg)

- **Preview player** — listen before applying, with non-destructive ReplayGain A/B
- **Health check** — codec-vs-extension mismatch, missing cover, deep scan for silence/clipping/transcode suspicion
- **Convert** — between MP3/FLAC/OGG/M4A/WAV/OPUS
- **Trim silence** — detect and cut leading/trailing silence
- **Repair** — remux broken streams
- **Normalize loudness** — EBU R128 measurement written as a ReplayGain tag (no re-encode)
- **Clean tags** — strip junk/duplicate tag blocks
- **Duplicate finder** — Chromaprint audio fingerprinting, fully offline
- **Undo** — destructive ops back up original bytes; one click restores

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

## Download

**Windows**: grab `AudioInk-amd64-installer.exe` from the [latest release](https://github.com/auazze/AudioInk/releases/latest) — it bundles FFmpeg/Chromaprint, adds the right-click "AudioInk Fix" context menu, and handles reinstall/uninstall.

**macOS / Linux**: no prebuilt binaries — build from source below (untested, see Platform support).

## Build from source

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

**FFmpeg note**: the audio toolbox shells out to `ffmpeg`, `ffprobe` and `fpcalc` (Chromaprint), resolved next to the executable first, then on PATH. The Windows installer bundles all three; a source build needs them installed yourself (`choco install ffmpeg` / `apt install ffmpeg`, fpcalc from [Chromaprint releases](https://github.com/acoustid/chromaprint/releases)) — without them the tag fixing works fine and the audio features just stay disabled.

### Windows installer

Requires [NSIS](https://nsis.sourceforge.io/Download) in PATH. The NSIS script bundles `ffmpeg.exe`, `ffprobe.exe` and `fpcalc.exe` from `build/bin/` — they are **not in git**, so stage them there first or the build fails on the `File` directives:

```bash
cp /path/to/ffmpeg.exe /path/to/ffprobe.exe /path/to/fpcalc.exe build/bin/
wails build --nsis
```

Creates `build/bin/AudioInk-amd64-installer.exe` — installs the app, adds right-click context menu for audio files ("AudioInk Fix"), handles reinstall/uninstall via registry.

### macOS installer (untested)

Build on a Mac, then run the install script:

```bash
wails build
bash build/darwin/install.sh
```

Copies `AudioInk.app` to `/Applications` and installs a Finder Quick Action (right-click menu). Detects existing installation, offers reinstall/uninstall.

### Linux installer (untested)

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
3. Review parsed results in the table — uncertain parses get a **Review** badge; double-click any row to edit artist/title inline, or fill the bulk Artist/Title fields
4. Optional: preview in the player, run a health check, convert/trim/repair/normalize from the audio toolbar
5. Click **Apply Tags** — choose **Save copies** or **Fix originals** (destructive ops are undoable)
6. For copies: find clean files in `AudioInk/` next to your originals

### CLI

```bash
AudioInk --fix file1.mp3 file2.flac ...
```

Runs headless: parses filenames, writes tags, renames files in place. Uncertain filenames open a single AudioInk-styled confirm window (all flagged files at once, prefilled with the parser's best guess) — confirm, edit, or skip. This is the mode the right-click context menu uses.

## License

[MIT](LICENSE)
