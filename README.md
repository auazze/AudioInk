# AudioInk

Lightweight desktop app that fixes audio file metadata. Parses filenames like `Artist - Song (feat. Someone).mp3` and writes correct ID3/Vorbis tags — artist, title, track number — then saves a clean copy with a proper filename.

## Why

Audio files often have filenames like `01. Tomoya Ohtani-Break Through It All (feat. Kellin Quinn).mp3` but empty metadata tags. Players (iPhone, AIMP, Spotify local files) show blank fields. AudioInk fixes this automatically.

## What It Does

| Input filename | Output filename | Artist tag | Title tag |
|---|---|---|---|
| `Tomoya Ohtani-Break Through It All (feat. Kellin Quinn).mp3` | `Tomoya Ohtani & Kellin Quinn - Break Through It All.mp3` | Tomoya Ohtani & Kellin Quinn | Break Through It All |
| `01. Queen - Bohemian Rhapsody (Live).flac` | `Queen - Bohemian Rhapsody (Live).flac` | Queen | Bohemian Rhapsody (Live) |
| `DJ Snake feat. Lil Jon - Turn Down for What (Remix).mp3` | `DJ Snake & Lil Jon - Turn Down for What (Remix).mp3` | DJ Snake & Lil Jon | Turn Down for What (Remix) |
| `Кино - Группа крови.mp3` | `Кино - Группа крови.mp3` | Кино | Группа крови |

### Parser handles

- **Separators**: ` - `, ` — `, ` – `, `_-_`, bare dashes
- **Featured artists**: `feat.`, `ft.`, `featuring` — merged with `&`
- **Track numbers**: `01.`, `01 -`, `1.`, `01_` — stripped
- **Extras**: `(Remix)`, `(Live)`, `(Acoustic)`, `[Explicit]` — kept in title
- **Copy suffixes**: `— копия`, `- Copy`, `(2)` — stripped
- **Confidence scoring**: high/medium/low — uncertain parses highlighted for review

### Safe by default

Originals are **never modified**. Output goes to `_AudioInk_output/` folder next to your files.

## Supported Formats

MP3, FLAC, OGG, M4A, WAV, WMA, OPUS

## Tech Stack

| Component | Choice |
|---|---|
| Language | Go |
| GUI | Wails v2 (Go backend + web frontend) |
| Frontend | Svelte |
| Audio tags | [go-taglib](https://pkg.go.dev/go.senan.xyz/taglib) (TagLib via Wasm, no CGo) |

## Build

Requirements: Go 1.23+, Node.js, [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build
wails build

# Dev mode with hot reload
wails dev
```

Binary output: `build/bin/AudioInk.exe`

## Usage

1. Launch `AudioInk.exe`
2. Drag & drop audio files onto the window, or use **Select files** / **Select folder**
3. Review parsed results in the table — double-click to edit artist/title
4. Click **Apply Tags**
5. Find clean files in `_AudioInk_output/` next to your originals
