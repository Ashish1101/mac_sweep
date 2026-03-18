# Mole GUI — Desktop GUI Wrapper for Mole CLI

## Context

[Mole](https://github.com/tw93/Mole) is a popular (40k+ stars) macOS system maintenance CLI tool written in **Go + Bash**. It handles disk cleaning, app uninstall, disk analysis, system optimization, and live monitoring — essentially a terminal-based CleanMyMac.

**Problem:** Terminal-based destructive operations (delete, clean, uninstall) are risky for non-technical users. A wrong command or missing `--dry-run` flag can permanently delete important files. There's no visual preview, no undo, and no risk indicators.

**Goal:** Build a GUI wrapper that makes Mole safe and accessible for non-tech users, without rewriting core logic.

---

## Mole CLI Analysis

### What Mole Does
- **Deep Cleaning:** Browser caches, app caches, system logs, temp files, language files
- **Smart Uninstall:** Removes app + preferences + launch agents + hidden remnants
- **Disk Analysis:** Visual file browser with size visualization, large file identification, interactive deletion
- **System Optimization:** Cache rebuilding, DNS flush, service refresh, memory optimization
- **Live Monitoring:** Real-time CPU, GPU, memory, disk, battery, network, Bluetooth metrics
- **Build Artifact Purging:** Clean project build artifacts (node_modules, target, etc.)

### Tech Stack
- **Go 1.25** — Core logic for `analyze` and `status` commands
  - `charmbracelet/bubbletea` — TUI framework
  - `charmbracelet/lipgloss` — Terminal styling
  - `shirou/gopsutil` — System metrics
- **Bash** — CLI entry point (`mole` script), clean/uninstall/optimize operations, library modules

### Repo Structure
```
Mole/
├── mole                    # Main Bash entry point (CLI router)
├── install.sh              # Installation script
├── go.mod / go.sum         # Go dependencies
├── Makefile                # Build configuration
├── lib/
│   ├── core/               # Shared utilities
│   │   ├── common.sh       # Common utilities and logging
│   │   ├── commands.sh     # Command definitions
│   │   ├── base.sh         # Platform detection, environment
│   │   ├── file_ops.sh     # Safe file deletion (safe_remove with rm -rf)
│   │   ├── app_protection.sh # Critical app whitelist management
│   │   ├── log.sh          # Structured logging with colors/icons
│   │   └── help.sh         # Help text
│   ├── clean/              # Cleanup functionality
│   ├── uninstall/          # App removal
│   ├── optimize/           # System optimization
│   ├── manage/             # App management
│   ├── check/              # Validation
│   └── ui/                 # Terminal user interface
├── cmd/
│   ├── analyze/            # Go — disk analysis & visualization
│   │   ├── main.go         # TUI interface (bubbletea)
│   │   ├── json.go         # JSON output structs (jsonOutput, jsonEntry)
│   │   ├── scanner.go      # Filesystem scanning
│   │   ├── cache.go        # Caching layer
│   │   ├── cleanable.go    # Cleanable file detection
│   │   ├── delete.go       # File deletion
│   │   ├── heap.go         # Priority queue for large files
│   │   ├── format.go       # Size/path formatting
│   │   └── constants.go    # Configuration constants
│   └── status/             # Go — system health monitoring
│       ├── main.go         # Dashboard TUI
│       ├── metrics.go      # Metrics collection framework
│       ├── metrics_battery.go
│       ├── metrics_bluetooth.go
│       ├── metrics_cpu.go
│       ├── metrics_disk.go
│       ├── metrics_gpu.go
│       ├── metrics_health.go
│       ├── metrics_memory.go
│       ├── metrics_network.go
│       ├── metrics_process.go
│       └── metrics_thermal.go
├── scripts/                # Automation helpers
├── tests/                  # Test suite
└── bin/                    # Compiled binaries
```

### CLI Commands
```bash
mo                    # Interactive menu
mo clean              # Deep cleanup (+ --dry-run, --debug, --whitelist)
mo uninstall          # Smart app removal (+ --dry-run)
mo optimize           # Refresh caches & services (+ --whitelist)
mo analyze            # Visual disk explorer (Go TUI)
mo status             # Live system health dashboard
mo purge              # Clean project build artifacts (+ --dry-run)
mo installer          # Find & remove installer files
mo touchid            # Configure Touch ID for sudo
mo completion         # Shell tab completion
mo update             # Update Mole
mo remove             # Uninstall Mole
```

### Safety Features in CLI
- `--dry-run` flag for preview
- Whitelist mechanism for protecting critical caches
- App protection rules (system-critical bundles)
- Structured logging

### Key Integration Points
- `cmd/analyze/json.go` — Already defines `jsonOutput` and `jsonEntry` structs with JSON tags (structured output ready)
- `cmd/status/metrics_*.go` — Modular metrics collection (CPU, memory, disk, battery, etc.)
- `lib/core/file_ops.sh` — `safe_remove()` function uses `rm -rf` (must intercept for GUI safety)
- `lib/core/app_protection.sh` — Protection rules for system-critical apps
- `MOLE_DRY_RUN=1` environment variable for dry-run mode

---

## Tech Stack: **Wails (Go + Svelte)**

### Comparison

| Criteria | Wails (Go) | Tauri (Rust) | Electron | SwiftUI | Flutter |
|---|---|---|---|---|---|
| Bundle size | ~10-15 MB | ~5-10 MB | ~150+ MB | ~2-5 MB | ~30-50 MB |
| RAM usage | Low-Medium | Low | High (200+ MB) | Lowest | Medium |
| Go interop | **Native (same lang)** | FFI/subprocess | subprocess | subprocess | subprocess |
| macOS integration | Good (WebKit) | Good (WebKit) | Poor | Excellent | Fair |
| Dev experience for Go devs | Lowest barrier | Medium (Rust) | Low (JS) | High (Swift) | Medium (Dart) |

### Why Wails Wins
1. **Same language as Mole's core** — Import `cmd/analyze` and `cmd/status` Go packages directly as libraries (type-safe, in-process), no subprocess needed
2. **Bash invocation is trivial** via Go's `os/exec`
3. **Small bundle** (~10-15 MB) — uses native macOS WebKit WebView, no bundled Chromium
4. **Svelte frontend** — smallest bundle sizes, excellent reactivity for real-time monitoring

### Why Not Others
- **Tauri:** Rust + Go FFI bridge is clumsy; forced into subprocess for all Go packages
- **Electron:** 150+ MB bundle for a cleanup tool is ironic
- **SwiftUI:** No Go code sharing, smaller data-viz ecosystem
- **Flutter:** Immature desktop support, no Go interop story

---

## Architecture

```
┌─────────────────────────────────────────────┐
│           Svelte Frontend (WebView)          │
│  Dashboard │ Clean │ Uninstall │ Analyze │...│
├─────────────────────────────────────────────┤
│              Wails Bridge (IPC)              │
│         (JSON-RPC over WebView binding)      │
├─────────────────────────────────────────────┤
│              Go Backend Service              │
│  ┌──────────┐ ┌──────────┐ ┌─────────────┐  │
│  │AnalyzeSvc│ │StatusSvc │ │ScriptRunner │  │
│  │(Go lib   │ │(Go lib   │ │(os/exec for │  │
│  │ import)  │ │ import)  │ │ bash scripts│  │
│  └──────────┘ └──────────┘ └─────────────┘  │
│       │            │              │           │
│  cmd/analyze  cmd/status    lib/*.sh scripts  │
└─────────────────────────────────────────────┘
```

### Communication Patterns

**Pattern A: Direct Go Library Import (analyze, status)**
- Extract core logic from `cmd/analyze` and `cmd/status` into importable packages
- `json.go` already defines structured output — confirms library-ready design
- Wails backend imports and exposes via bound methods

**Pattern B: Subprocess Execution (clean, uninstall, optimize, purge)**
- Run bash scripts via `os/exec`
- Always `--dry-run` first → parse output → present preview → execute on confirmation
- Parse stdout with regex (short-term) or `--json` flag (propose upstream)

**Pattern C: Real-time Event Streaming**
- Long operations use Wails `runtime.EventsEmit` for progress updates
- Frontend subscribes via `EventsOn`
- 2-second polling for system monitoring

### Output Parsing Strategy
- **Phase 1:** Regex parsing of human-readable stdout (scripts use consistent patterns like `[DRY RUN] Would remove: /path (size)`)
- **Phase 2:** Propose upstream `--json` flag / `MO_OUTPUT=json` env var for structured output

---

## UI/UX Design

### Navigation: Sidebar with 6 primary modules

### Screen 1: Dashboard / Home
```
┌──────────┬──────────────────────────────────────┐
│          │  System Health Score: 87/100          │
│ ● Home   │                                       │
│ ○ Clean  │  ┌─────────┐ ┌─────────┐ ┌────────┐  │
│ ○ Uninst │  │Disk Used│ │ Memory  │ │Battery │  │
│ ○ Analyze│  │ 78%     │ │  6.2GB  │ │  92%   │  │
│ ○ Monitor│  └─────────┘ └─────────┘ └────────┘  │
│          │                                       │
│ ─────── │  Quick Actions:                        │
│ Settings │  [Quick Clean]  [Scan Disk]            │
│          │                                       │
│          │  Last cleaned: 3 days ago (2.1 GB)    │
│          │  Reclaimable space: ~4.7 GB           │
└──────────┴──────────────────────────────────────┘
```

### Screen 2: Clean Module (Two-step: Scan → Execute)

**Step 1 — Scan Results:**
```
┌──────────┬──────────────────────────────────────┐
│          │  Clean                    [Scan]      │
│          │                                       │
│          │  Category          Size   Risk  [✓]   │
│          │  ─────────────────────────────────     │
│          │  Browser Caches    1.2GB  LOW   [✓]   │
│          │  App Caches        890MB  LOW   [✓]   │
│          │  System Logs       340MB  MED   [✓]   │
│          │  Dev Tool Caches   2.1GB  LOW   [✓]   │
│          │  Device Backups    4.0GB  HIGH  [ ]   │
│          │                                       │
│          │  Total selected: 4.53 GB              │
│          │  [Expand Category] shows file list    │
│          │                                       │
│          │       [Cancel]  [Clean Selected]      │
└──────────┴──────────────────────────────────────┘
```

**Step 2 — Confirmation:**
```
┌─────────────────────────────────────┐
│  Confirm Cleanup                    │
│                                     │
│  Will permanently remove:           │
│  • 247 files across 4 categories    │
│  • Total: 4.53 GB                   │
│                                     │
│  Items will be moved to Trash       │
│  (recoverable for 30 days)          │
│                                     │
│  [Cancel]        [Move to Trash]    │
└─────────────────────────────────────┘
```

### Screen 3: Uninstall Module
```
┌──────────┬──────────────────────────────────────┐
│          │  Uninstall          [Search...]       │
│          │                                       │
│          │  Sort: [Name] [Size] [Last Used]      │
│          │                                       │
│          │  [ ] Slack          180MB  Today       │
│          │  [ ] Docker Desktop  2.1GB  3d ago    │
│          │  [✓] GarageBand     1.8GB  Never      │
│          │  [ ] Keynote        540MB  30d ago    │
│          │                                       │
│          │  Protected: Finder, Safari (12 apps)  │
│          │                                       │
│          │  Selected: GarageBand (1.8 GB)        │
│          │  + 23 remnant files (140 MB)          │
│          │                                       │
│          │  [Cancel]  [Preview Removal]          │
└──────────┴──────────────────────────────────────┘
```

### Screen 4: Analyze Module (Disk Visualization)
```
┌──────────┬──────────────────────────────────────┐
│          │  Disk Analysis      [Path: /]         │
│          │                                       │
│          │  ┌──────────────────────────────────┐ │
│          │  │          TREEMAP VIEW             │ │
│          │  │  ┌────────┐┌─────┐┌──┐           │ │
│          │  │  │ System ││Users││  │           │ │
│          │  │  │ 45GB   ││120GB││  │           │ │
│          │  │  └────────┘└─────┘└──┘           │ │
│          │  └──────────────────────────────────┘ │
│          │                                       │
│          │  /Users/you (120 GB)                   │
│          │  ├── Documents   45.2 GB              │
│          │  ├── Library     32.1 GB              │
│          │  ├── Downloads   28.4 GB              │
│          │                                       │
│          │  [Delete Selected] [Move to Trash]    │
└──────────┴──────────────────────────────────────┘
```

### Screen 5: System Monitor
```
┌──────────┬──────────────────────────────────────┐
│          │  System Monitor                       │
│          │                                       │
│          │  CPU ████████░░░░ 62%    4 cores      │
│          │  MEM ██████░░░░░░ 48%    8/16 GB      │
│          │  DSK ██████████░░ 82%    200/256 GB   │
│          │  BAT ████████████ 96%    Plugged In   │
│          │                                       │
│          │  CPU History (last 60s)               │
│          │  [sparkline chart]                    │
│          │                                       │
│          │  Top Processes:                        │
│          │  Chrome        12.3%  1.2GB           │
│          │  Docker        8.1%   890MB            │
│          │                                       │
│          │  Health Score: 87/100  [Details]      │
└──────────┴──────────────────────────────────────┘
```

### Screen 6: Settings
- Visual whitelist editor with native file picker
- Protected apps list (read-only for system-critical)
- Operation history browser with search/filter
- Theme (light/dark/system)
- Trash retention period config
- Export audit log as CSV

---

## Safety Features (Core Value Proposition)

### 1. Trash-First Deletion (most critical improvement)
CLI uses `rm -rf` via `safe_remove()`. GUI replaces with macOS Trash:
```go
// Use macOS Finder to move to Trash (recoverable)
osascript -e 'tell app "Finder" to delete POSIX file "/path"'
// Or use `trash` CLI: brew install trash
```

### 2. Mandatory Dry-Run Preview
Every destructive operation: **Scan → Preview → Confirm → Execute → Report**
No way to skip the preview step.

### 3. Risk Classification
- **LOW** (caches, temp files): Green badge, checked by default
- **MEDIUM** (system logs): Yellow badge, checked with info tooltip
- **HIGH** (backups, large files): Red badge, **unchecked by default**, extra confirmation

### 4. Undo/Restore
- Session log in `~/.config/mole-gui/sessions/`
- "Recent Operations" panel with per-item "Restore" button
- Auto-cleanup after configurable retention (default 30 days)

### 5. Audit Trail
- All operations logged to `~/.config/mole-gui/audit.log` (JSON)
- Exportable as CSV

### 6. Native Privilege Escalation
macOS-native dialog instead of terminal sudo:
```
osascript -e 'do shell script "..." with administrator privileges'
```

### 7. Confirmation Dialog Hierarchy
- **Single file:** Inline confirmation
- **Batch (LOW risk):** Standard modal with count + size
- **Batch (includes HIGH risk):** Two-step — summary modal, then type-to-confirm
- **App uninstall:** Dedicated confirmation with app name, remnant paths, total size

---

## Implementation Phases

### Phase 1: Foundation + Analyze (Weeks 1-3)
- Initialize Wails v2 + Svelte project
- Sidebar navigation and routing
- Import `cmd/analyze` as Go library, expose via Wails bindings
- Treemap visualization (D3.js) + file browser
- Trash-based deletion in `backend/safety.go`
- **Deliverable:** App that scans, visualizes, and safely deletes files

### Phase 2: Monitor + Dashboard (Weeks 4-5)
- Import `cmd/status/metrics_*.go` as library
- Real-time polling backend with Wails event streaming (2s interval)
- Monitor UI: gauges, sparklines, process list
- Dashboard with health score and quick stats
- **Deliverable:** Live monitoring and informative dashboard

### Phase 3: Clean Module (Weeks 6-8)
- `ScriptRunner` for bash subprocess management
- Dry-run output parsing (regex initially)
- Category-based preview UI with risk badges and checkboxes
- Selective execution with exclusions
- Progress streaming and confirmation dialogs
- Whitelist visualization
- **Deliverable:** Full clean workflow with preview and trash-based deletion

### Phase 4: Uninstall Module (Weeks 9-10)
- App scanner with icon extraction from `.app` bundles
- Remnant scanning and preview
- Protection system integration (read `app_protection.sh` patterns)
- Uninstall confirmation flow
- **Deliverable:** Visual app uninstaller with remnant preview

### Phase 5: Polish + Settings (Weeks 11-13)
- Settings panel: whitelist editor, theme, history
- Audit logging and undo/restore
- Optimize + purge module wrappers
- Menu bar tray icon, notifications
- First-run onboarding flow
- Error handling and performance optimization

### Phase 6: Distribution (Week 14)
- Code signing + notarization
- DMG installer
- Homebrew cask formula: `brew install --cask mole-gui`
- Auto-update mechanism

---

## Key Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Go packages coupled to bubbletea TUI | Refactor to extract core logic into `pkg/` packages. `json.go` shows structured output already exists. Contribute upstream. |
| Bash output parsing is fragile | Regex parsing initially; propose upstream `--json` flag. Architecture supports swapping parsers. |
| Sudo requirement | macOS-native privilege escalation dialog via `osascript` |
| Mole CLI updates break parsing | Pin to specific version, integration tests against output format |
| Large scans block UI | Go goroutines + Wails events for streaming. Virtual scrolling for large lists. |

---

## Project Structure

```
mole-gui/
├── main.go                 # Wails app entry
├── app.go                  # App lifecycle + Wails bindings
├── backend/
│   ├── analyze.go          # Import cmd/analyze packages
│   ├── status.go           # Import cmd/status packages
│   ├── scripts.go          # Bash script runner (clean/uninstall/optimize)
│   ├── safety.go           # Trash-based deletion, audit logging
│   └── config.go           # Settings management
├── frontend/
│   ├── src/
│   │   ├── App.svelte
│   │   ├── lib/
│   │   │   ├── Sidebar.svelte
│   │   │   ├── Dashboard.svelte
│   │   │   ├── Clean.svelte
│   │   │   ├── Uninstall.svelte
│   │   │   ├── Analyze.svelte
│   │   │   ├── Monitor.svelte
│   │   │   └── Settings.svelte
│   │   ├── components/     # Shared UI components
│   │   └── stores/         # Svelte stores for state
│   └── package.json
├── wails.json
└── go.mod
```

---

## Verification Plan

- `wails dev` for hot-reload development
- Test each module against Mole CLI output
- Verify trash-based deletion (file appears in macOS Trash)
- Test dry-run parsing against all Mole command outputs
- Test privilege escalation dialog for sudo operations
- Cross-test with macOS 12+
- Integration tests pinned to specific Mole version
