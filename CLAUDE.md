# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Build
- Main build: `go build -o gacha.exe ./src`
- Portable Windows build (no Go required): `pwsh -File scripts\build_portable_win.ps1`
- Package distribution ZIP: `scripts\package_dist.ps1` or `scripts\package_dist.bat`

### Testing
- Automated tests (Linux/WSL): `python3 test/auto/run_tests.py` (requires Linux binary `gacha`)  
- Manual test with 20 users: `test\manual\test_invoke_20_users.ps1` or `.bat`
- Test specifications: `test/specs/test_spec.md`

### Development Scripts
- Start API server: `scripts\serve_api.bat [port]` (default: 3010)
- Stop API server: `scripts\stop_api.bat`
- Reset system: `scripts\reset.bat` (creates backup, then initializes)
- Create new session: `scripts\new_session.bat` (backup + reset + regenerate index)
- Restore from backup: `scripts\restore.bat BACKUP_NAME`

## Architecture Overview

This is a Go-based Twitch gacha (lottery) aggregation system designed to:
1. Accept external triggers from streaming software (たぬえさ)
2. Update winner data in real-time
3. Generate OBS-compatible browser source displays
4. Optionally notify Discord via webhooks

### Core Components

**Main Binary (`src/main.go`)**
- Single Go file containing all application logic (~1400+ lines)
- Handles CLI commands, HTTP API server, data persistence, Discord integration
- Core data structures: `User`, `State`, `Settings`, `Event`, `Session`

**Data Flow**
1. External trigger: `gacha.exe "username" hitFlag` (0=win, 1=jackpot)
2. Updates `data/current.json` with winner data
3. Generates `data/data.js` for OBS browser source consumption
4. Logs events to `logs/` directory (optional)
5. Sends Discord notifications (optional)

**Key Directories**
- `src/` - Single Go source file
- `data/` - Current state, generated JS, Discord mapping, session info
- `public/` - OBS browser source HTML/CSS/JS
- `backups/` - Historical snapshots created during resets
- `logs/` - Event logs and application logs
- `scripts/` - Windows batch/PowerShell utilities

### Configuration

**Primary config: `setting.json`**
- `autoServe`: Auto-start API server on gacha.exe execution
- `serverPort`: API server port (default 3010)
- `eventJsonLog`: Enable per-event JSON logging
- `discordEnabled`: Enable Discord webhook notifications
- `discordNewMessagePerSession`: Create new Discord message per session
- `discordArchiveOldSummary`: Archive old Discord messages with timestamp

**Environment variables (`.env.local`)**
- `DISCORD_NOTIFY`: Enable Discord notifications (1/true/yes/on)
- `DISCORD_WEBHOOK_URL`: Discord webhook URL for notifications

### API Endpoints (when server running)
- `POST /reset` - Reset system (backup current state, initialize fresh)
- `POST /gen-backup-index` - Regenerate backup index for history display

### OBS Integration
- Load `public/index.html` as browser source in OBS
- Automatically refreshes `data/data.js` every few seconds
- History viewing via dropdown (reads `backups/index.js`)
- UI reset option (requires API server running)

### Discord Integration
- Summary-style notifications (consolidated message per session)
- Format: `[Gif] username` for jackpots, `[Ilst] username` for regular wins
- Maintains message mapping in `data/discord_map.json`
- Archives old messages when starting new sessions

## Go Module Information
- Module name: `twitch-tanuesa-result`
- Go version: 1.21
- No external dependencies (uses only standard library)