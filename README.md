# box-link

[English](README.md) | [中文](docs/README_zh.md)

`box-link` is a macOS-focused tool for managing a direct network link to a box device.

It ships one Go binary with two entry modes:

- CLI mode for terminal users
- Local Web UI mode for users who prefer clicking buttons

The project keeps interface detection, logging, diagnostics, packaging, and release flow in one codebase.

See `docs/design.md` for architecture details and `docs/todo.md` for the execution plan.

## Requirements

- macOS
- Go installed for local development
- `sudo` for `on`, `off`, and usually `ui` when the tool needs to change network settings

## Commands

Run this any time:

```bash
box-link --help
```

Available commands:

- `info`: show the active config file path and the effective settings
- `iface`: resolve the interface that should be used for the direct box link
- `status`: print interface, route, and ping output
- `doctor`: print a concise diagnostic summary
- `on`: configure the selected interface with the host IP and test box reachability
- `off`: restore the selected interface to DHCP
- `ui`: start the local Web UI server
- `version`: print the current version

## Configuration

`box-link` now uses a default config file at:

```text
~/.box-link/config.toml
```

The file is created automatically on first run if it does not exist.

Starter content:

```toml
# box-link configuration
config_path = "/Users/<you>/.box-link/config.toml"
iface = ""
host_ip = "192.168.10.3"
box_ip = "192.168.10.2"
netmask = "255.255.255.0"
hardware_port_pattern = "AX88179A"
log_level = "DEBUG"
listen_addr = "127.0.0.1:18888"
```

Notes:

- `config_path` is written into the file and also shown by `box-link info` and the Web UI
- `iface = ""` means `Iface override` is `(auto)`
- if you want to force a specific interface, set `iface = "en7"` or similar

Supported environment variables still work and override file values:

```bash
BOX_CONFIG_PATH
BOX_IFACE
BOX_HOST_IP
BOX_IP
BOX_NETMASK
BOX_HARDWARE_PORT_PATTERN
BOX_LOG_LEVEL
BOX_LISTEN_ADDR
```

Interface resolution order:

1. `iface` / `BOX_IFACE` if explicitly set and valid
2. `networksetup -listallhardwareports` matching `hardware_port_pattern` / `BOX_HARDWARE_PORT_PATTERN`
3. An `en*` interface already holding `host_ip` / `BOX_HOST_IP`
4. The highest-ranked active external `en*`
5. The highest-numbered external `en*` fallback

## Typical Usage

CLI flow:

```bash
box-link info
box-link iface
sudo box-link on
box-link status
sudo box-link off
```

Web UI flow:

```bash
sudo box-link ui
```

Then open:

```text
http://127.0.0.1:18888
```

The Config panel in the UI shows:

- config file path
- iface override
- host IP
- box IP
- netmask
- hardware match pattern
- log level
- listen address

## Development

The project uses `just` for local workflows:

```bash
just fmt
just build
just test
just run status
just package
just package-release
just package-pkg
just homebrew-formula
just homebrew-tap
```

Notes:

- `just package` builds a tarball for the current host platform
- `just package-release` builds release tarballs for `darwin/arm64` and `darwin/amd64`
- `just package-pkg` builds a macOS `.pkg` installer on a Mac host
- `just homebrew-formula` generates a Homebrew formula from the current release artifacts
- `just homebrew-tap` writes the formula directly into a tap repo's `Formula/` directory
- both commands call the same script: `packaging/package.sh`
- `./scripts/check.sh` also supports short profile aliases: `--pre-commit`, `--ci`, and `--full`

## Packaging

Build a local package:

```bash
just package
```

Build the release target set:

```bash
just package-release
```

Build a macOS installer package:

```bash
just package-pkg
```

Generate a Homebrew formula:

```bash
just homebrew-formula your-org/box-link
```

Write directly into a tap repo:

```bash
just homebrew-tap your-org/box-link ../homebrew-tools
```

Artifacts land in `dist/`:

- `box-link-<version>-<goos>-<goarch>.tar.gz`
- `box-link-<version>.pkg`
- `checksums.txt`

Each archive contains:

- `box-link`
- `install.sh`
- `README.md`

After extracting an archive:

```bash
./install.sh
```

## Release Flow

Tag format:

```text
v<major>.<minor>.<patch>
```

Local release helper:

```bash
just release 0.1.0
```

GitHub Actions will then:

- validate the tag format
- build release tarballs on Ubuntu
- generate and attach a Homebrew formula
- build and attach a macOS `.pkg` installer
- generate release notes with `git-cliff`
- publish everything in one GitHub Release

## Friendly Distribution

For non-developer users, two next-step distribution paths make sense:

### Homebrew Tap

Best for technical macOS users who want a one-command install:

```bash
brew tap <org>/tools
brew install box-link
```

Recommended setup:

- create a separate tap repo such as `homebrew-tools`
- publish the release tarballs from GitHub Releases
- generate a formula with `just homebrew-formula <owner/repo>`
- or write directly into a tap repo with `just homebrew-tap <owner/repo> <tap-dir>`

A starter formula template lives at `packaging/homebrew/box-link.rb`.

### macOS Installer Package

Best for non-technical users who expect a double-click installer.

Recommended flow:

- keep `just package-release` as the binary packaging step
- produce a `.pkg` with `just package-pkg`
- sign and notarize the installer before wider distribution

Suggested commands:

```bash
just package-pkg com.example.box-link
```

A starter checklist lives at `packaging/pkg/README.md`, and the actual builder lives at `packaging/build-pkg.sh`.

Generate a Homebrew formula for a release repo:

```bash
just homebrew-formula your-org/box-link
```
