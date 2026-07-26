# MetalTracker

Private, local desktop app for tracking a precious metal portfolio (gold and silver bars & coins). Built with [Wails](https://wails.io) (Go + React).

## Features

- PIN-encrypted vault on your device
- One-time recovery key at setup
- Unit-level holdings with groups, sell/edit, and P&L
- Spot prices via our API or MetalpriceAPI (BYOK). Alternatively you can host your own instance of our Middleman API.

## Setup

1. Install [Go](https://go.dev/), [Node.js](https://nodejs.org/), and [Wails](https://wails.io/docs/gettingstarted/installation).
2. From the repo root:

```bash
wails dev
```

3. On first launch, create a **6-digit PIN**. Copy and store the recovery key offline.
4. Open **Settings** if you want to use your own [MetalpriceAPI](https://metalpriceapi.com/) key.
5. Use **Add** to record a purchase.

### Building

Embed a version:

```bash
# Windows
wails build -nsis

# macOS / Linux
wails build

wails build -ldflags "-X MetalTracker/internal/version.Version=v0.0.0-dev"
```

Build for specific platform:

```bash
wails build -platform windows/amd64
wails build -platform darwin/arm64
wails build -platform darwin/amd64
wails build -platform linux/amd64
```

Linux builds need GTK/WebKit packages (Debian/Ubuntu 24.04+):

```bash
sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev
wails build -platform linux/amd64 -tags webkit2_41
```

macOS code signing / notarization is optional and not required for local builds.

## Install (releases)

Official builds are published on [GitHub Releases](https://github.com/Marijn17s/MetalTracker/releases) when a version tag is pushed (e.g. `v1.2.0`).

| Platform | Asset | Notes |
|----------|--------|--------|
| Windows | `MetalTracker-Windows-x64-Setup.exe` | NSIS installer (also used by in-app updates) |
| macOS Apple Silicon | `MetalTracker-macOS-AppleSilicon.dmg` | Drag the app to Applications (also used by in-app updates) |
| macOS Intel | `MetalTracker-macOS-Intel.dmg` | Same |
| Linux | `MetalTracker-Linux-x64.tar.gz` | Extract and run; needs WebKitGTK 4.1 at runtime |

Verify downloads against `SHA256SUMS` on the same release.

### Releasing a new version (maintainers)

1. Merge the work you want to ship to the main branch.
2. Create and push a tag: `vMAJOR.MINOR.PATCH`
3. GitHub Actions builds Windows / macOS / Linux, uploads assets + `SHA256SUMS`, and creates the GitHub Release.
4. Do **not** change version strings in source for each release - CI embeds the tag via `-ldflags`.

Prerelease tags with a hyphen (e.g. `v1.3.0-beta.1`) can be published manually; the in-app checker uses GitHub’s “latest” release (non-prerelease).

## Updates

In **Settings -> About & updates**:

1. **Check for updates** queries the latest GitHub Release.
2. If newer, review notes and choose **Install update**.
3. The app downloads the portable asset for your OS, verifies the SHA-256 from `SHA256SUMS`, replaces the running binary, and restarts.

## PIN & recovery

- PIN must be **exactly 6 digits**.
- If you forget the PIN, use **Forgot PIN?** with your recovery key, then set a new 6-digit PIN. The recovery key stays valid.
- If an older vault used a longer PIN, unlock is no longer possible with that PIN - recover with the recovery key and set a 6-digit PIN.

## Backup & migrate

In **Settings -> Backup**:

- **Backup** - enter PIN, pick a save location (encrypted `.mtbackup`)
- **Verify backup** - enter recovery key, pick a backup to verify (no changes)
- **Restore** - enter recovery key, pick a backup (replaces this vault, then locks)
- **Save kit** - printable recovery-key HTML

New Device: Backup -> install MetalTracker -> Restore -> unlock -> check holdings.

Vault files live under the OS user config directory (e.g. `%AppData%\MetalTracker` on Windows).

## Price sources

| Source | Notes |
|--------|--------|
| MetalpriceAPI | API key stored encrypted in the vault; Requests use `unit=kilogram` (requires paid plan). |
| Middleman | Shared cache HTTP API (`middlemanBaseUrl`). New vaults default to use the Middleman at `https://metaltracker.moose-vimba.ts.net`; you can switch source or point at your own host. Docker: see [`middleman/README.md`](middleman/README.md). |

Spot rates are stored and valued **per kilogram** internally; Settings lets you display spot in g / troy oz / kg.

## Development

- Backend: `internal/` (domain, storage, price, security, service)
- Frontend: `frontend/src/`
- Tests: build the frontend once (`cd frontend && npm run build`), then `go test ./...` (root package embeds `frontend/dist`). For package tests only: `go test ./internal/...`