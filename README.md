# nd

<!-- rumdl-disable MD044 -->
[![CI](https://github.com/armstrongl/nd/actions/workflows/ci.yml/badge.svg)](https://github.com/armstrongl/nd/actions/workflows/ci.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/armstrongl/nd)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/armstrongl/nd)](https://github.com/armstrongl/nd/releases)
<!-- rumdl-enable MD044 -->

> [!WARNING]
> nd is still in alpha and under active development. Please let me know if you encounter issues 🙂

Manage coding agent assets (skills, agents, commands, rules, and more) across tools like Claude Code with symlink-based deployment.

![nd demo](assets/napoleon-dynamite-skills.gif)

## What it does

- **Register sources:** Point nd at local directories or git repos containing agent assets
- **Deploy assets:** Create symlinks from agent config directories to source assets, keeping everything in sync
- **Switch profiles:** Group assets into named profiles and switch between them instantly
- **Save snapshots:** Capture and restore deployment states as point-in-time snapshots

## Install

### Homebrew (macOS/Linux)

```shell
brew install --cask armstrongl/tap/nd
```

### Go install

```shell
go install github.com/armstrongl/nd@latest
```

### GitHub releases

Download pre-built binaries from [Releases](https://github.com/armstrongl/nd/releases).

### Build from source

```shell
git clone https://github.com/armstrongl/nd.git
cd nd
go build -o nd .
```

## Quick start

```shell
# Initialize nd configuration
nd init

# Register an asset source (local directory or git repo)
nd source add ~/my-assets
# or: nd source add github-user/asset-repo

# See available assets
nd list

# Deploy an asset
nd deploy skills/greeting

# Check deployment status
nd status
```

## Commands

| Command | Description |
|---------|-------------|
| `nd init` | Initialize nd configuration |
| `nd source add` | Register a local directory or git repo as an asset source |
| `nd source remove` | Remove a registered source |
| `nd source list` | List all registered sources |
| `nd list` | List all available assets from all sources |
| `nd deploy` | Deploy assets by creating symlinks |
| `nd remove` | Remove deployed assets |
| `nd status` | Show deployment status and health |
| `nd pin` / `nd unpin` | Pin/unpin assets to persist across profile switches |
| `nd sync` | Repair symlinks and pull git sources |
| `nd doctor` | Health check: validate config, sources, deployments |
| `nd profile create` | Create a named profile (asset collection) |
| `nd profile switch` | Switch between profiles |
| `nd profile deploy` | Deploy all assets from a profile |
| `nd profile delete` | Delete a profile |
| `nd profile add-asset` | Add an asset to an existing profile |
| `nd profile list` | List all profiles |
| `nd snapshot save` | Save current deployments as a snapshot |
| `nd snapshot restore` | Restore deployments from a snapshot |
| `nd snapshot list` | List all snapshots |
| `nd snapshot delete` | Delete a snapshot |
| `nd settings edit` | Open config file in your editor |
| `nd uninstall` | Remove all nd-managed symlinks |
| `nd version` | Print version information |
| `nd completion` | Generate shell completions (bash, zsh, fish) |

Run any command with `--help` for detailed usage, or see the full [Command Reference](docs/reference/nd.md).

Many commands support **interactive mode**: run without arguments to get a picker. Use `--json` for scripted output and `--yes` to skip confirmations.

## Configure

nd uses a YAML config file at `~/.config/nd/config.yaml`. Key settings:

```yaml
version: 1
default_scope: global       # or "project"
default_agent: claude-code
symlink_strategy: absolute  # or "relative"
```

See the full [configuration guide](docs/guide/configuration.md).

## Documentation

- [How nd works](docs/guide/how-nd-works.md): What happens on disk when you deploy
- [Get started](docs/guide/getting-started.md): Install to first deploy in 5 minutes
- [User guide](docs/guide/user-guide.md): Core workflows: sources, deploying, syncing
- [Profiles & snapshots](docs/guide/profiles-and-snapshots.md): Advanced workflow management
- [Configuration](docs/guide/configuration.md): Full config reference
- [Creating sources](docs/guide/creating-sources.md): Build your own asset library
- [Command reference](docs/reference/nd.md): Auto-generated from source

## Contribute

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and PR guidelines.

For architecture details, see [ARCHITECTURE.md](ARCHITECTURE.md).

## License

[MIT](LICENSE)
