# MacSweep

A macOS disk cleaner app built with [Wails](https://wails.io/) (Go backend + Svelte frontend). Analyze disk usage, clean junk files, monitor system resources, and manage operation history with restore support.

## Installation

Install via [Homebrew](https://brew.sh/):

```bash
brew tap Ashish1101/tap
brew install --cask macsweep
```

Or download the latest `.app` bundle directly from [Releases](https://github.com/Ashish1101/mac_sweep/releases).

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Development

Run in live development mode with hot reload:

```bash
$(go env GOPATH)/bin/wails dev
```

This starts a Vite dev server for the frontend with hot reload. A browser dev server is also available at http://localhost:34115 where you can call Go methods from devtools.

## Building

Build a production `.app` bundle:

```bash
$(go env GOPATH)/bin/wails build
```

The built application will be at `build/bin/macsweep.app`.

To run the built app:

```bash
open build/bin/macsweep.app
```
