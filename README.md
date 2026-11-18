# Dashcam Studio

A desktop player for 3-channel dashcam cards, with a Hebrew (RTL) interface. It plays the Front, Rear
and Cabin cameras together, with the GPS route, favorites and ZIP export.

## Run it

1. Plug in the dashcam card.
2. Open the file for your computer. There is nothing to install.

| Computer | File | How to open it |
|---|---|---|
| Windows 10/11 | `dashcam-studio-<ver>-windows-x64.exe` | Double-click it. |
| Mac | `dashcam-studio-<ver>-macos-universal.zip` | Unzip it, then double-click **Dashcam Studio.app**. The first time, right-click it and choose **Open**, because the app isn't notarized. |
| Linux | `dashcam-studio-<ver>-linux-x64.tar.gz` | Extract it, then run `./dashcam-studio`. |

3. The app finds the card by itself. Click it on the welcome screen, or choose a folder.

To quit, close the window.

The app only reads the card and never changes anything on it. On Mac and Linux it opens in a
Chrome, Edge or Brave window, or in your normal browser if none of those is installed.

**Optional:** to get clip thumbnails, put `ffmpeg` next to the app or install it on your system.
Everything else works without it.

## Build it yourself

You need [Go](https://go.dev/dl/) 1.27 or newer and [Node.js](https://nodejs.org/) 20 or newer.
From the project folder, run:

```
go run ./tools/build
```

The files for Windows, Mac and Linux appear in `dist/`. Any computer can build all three.

## Command line (optional)

```
dashcam-studio E:\                 # open a card or folder directly
dashcam-studio --help              # all options (--port, --ui, --debug, ...)
```

## Keyboard

| Key | Action | Key | Action |
|---|---|---|---|
| Space | play / pause | ← → | back / forward 5 s |
| P / N | previous / next clip | 1 2 3 | Front / Rear / Cabin as main view |
| F | fullscreen | Esc | leave fullscreen / close a dialog |
| `*` | star the current clip | D | download the clip as a ZIP |
| S | snapshot | E | disconnect / eject the card |
| ? | all shortcuts | | |

Shortcuts also work with a Hebrew keyboard layout.

## Where the app keeps its files

- **Favorites:** `%APPDATA%\dashcam-studio` on Windows, `~/Library/Application Support/dashcam-studio` on Mac,
  `~/.config/dashcam-studio` on Linux.
- **Thumbnails and the log:** `%LOCALAPPDATA%\dashcam-studio` on Windows, `~/Library/Caches/dashcam-studio` on Mac,
  `~/.cache/dashcam-studio` on Linux.

## For developers

```
go run ./tools/build dev       # live-reload dev loop: Vite (UI) + Go API, opens a browser
go run ./tools/build build     # quick build for this computer only, without tests
go run ./tools/build test      # all tests
```

The project has three parts:

- **Go server:** in `cmd/` and `internal/`.
- **React UI:** in `web/`. All user-facing text is in `web/src/i18n/he.ts`.
- **Build pipeline:** in `tools/build/`.

Detailed notes for working on the code are in `.claude/skills/dashcam-studio-dev/SKILL.md`.
