# MetalTracker

Private, local-first desktop app for tracking a precious-metal portfolio (gold and silver bars & coins). Built with [Wails](https://wails.io) (Go + React).

## Features

- PIN-encrypted vault on your device (exactly **6 digits**)
- One-time recovery key at setup
- Unit-level holdings with groups, sell/edit, and P&L
- Spot prices via MetalpriceAPI (local cache) or optional Middleman URL
- Display currency with FX conversion, spot units in g / troy oz / kg
- In-app Help

## Setup

1. Install [Go](https://go.dev/), [Node.js](https://nodejs.org/), and [Wails](https://wails.io/docs/gettingstarted/installation).
2. From the repo root:

```bash
wails dev
```

3. On first launch, create a **6-digit PIN**. Copy and store the recovery key offline.
4. Open **Settings** if you want your own [MetalpriceAPI](https://metalpriceapi.com/) key; new vaults already use the default Middleman URL.
5. Use **Add** to record a purchase.

### Building (local)

Embed a version (optional for local smoke builds):

```bash
# Windows
wails build -ldflags "-X MetalTracker/internal/version.Version=v0.0.0-dev"

# macOS / Linux
wails build -ldflags "-X MetalTracker/internal/version.Version=v0.0.0-dev"
```

Cross-platform examples:

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

macOS code signing / notarization is optional and not required for ad-hoc local builds. CI publishes an unsigned `.dmg` for install.

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

1. Merge the work you want to ship to the default branch.
2. Create and push a semver tag: `vMAJOR.MINOR.PATCH` (lazygit: tag commit -> Tags tab -> `P` to push).
3. GitHub Actions builds Windows / macOS / Linux, uploads assets + `SHA256SUMS`, and creates the GitHub Release.
4. Do **not** change version strings in source for each release - CI embeds the tag via `-ldflags`.

Prerelease tags with a hyphen (e.g. `v1.3.0-beta.1`) can be published manually; the in-app checker uses GitHub’s “latest” release (non-prerelease).

## Updates

In **Settings -> About & updates**:

1. **Check for updates** queries the latest GitHub Release.
2. If newer, review notes and choose **Install update**.
3. The app downloads the portable asset for your OS/arch, verifies the SHA-256 from `SHA256SUMS`, replaces the running binary, and restarts.

Auto-update requires a **writable** install path (portable exe / user-owned copy). Installs under protected directories (e.g. `Program Files`) may need a manual reinstall from the release page.

## PIN & recovery

- PIN must be **exactly 6 digits**.
- If you forget the PIN, use **Forgot PIN?** with your recovery key, then set a new 6-digit PIN.
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
| MetalpriceAPI | API key stored encrypted in the vault; quotes cached locally (~6 hours). Requests use `unit=kilogram` (paid plan). |
| Middleman | Shared cache HTTP API (`middlemanBaseUrl`). New vaults default to Middleman at `https://metaltracker.moose-vimba.ts.net`; you can switch source or point at your own host. Docker: see [`middleman/README.md`](middleman/README.md). |

Spot rates are stored and valued **per kilogram** internally; Settings still lets you display spot in g / troy oz / kg.

## Help in the app

Open **Help** in the sidebar for vault, PIN, recovery key, prices, currency, and update notes.

## Development

- Backend: `internal/` (domain, storage, price, security, service)
- Frontend: `frontend/src/`
- Tests: build the frontend once (`cd frontend && npm run build`), then `go test ./...`  
  (root package embeds `frontend/dist`). For package tests only: `go test ./internal/...`
