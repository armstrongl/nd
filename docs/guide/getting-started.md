# Getting Started

This guide takes you from zero to your first deployed asset in about 5 minutes.

## 1. Install nd

Choose your preferred method:

```bash
# Homebrew (macOS/Linux)
brew install --cask armstrongl/tap/nd

# Go install
go install github.com/armstrongl/nd@latest

# Or build from source
git clone https://github.com/armstrongl/nd.git && cd nd && go build -o nd .
```

Verify the installation:

```bash
nd version
```

## 2. Initialize

Create the nd configuration directory and default config:

```bash
nd init
```

This creates `~/.config/nd/config.yaml` with sensible defaults and sets up directories for profiles, snapshots, and state.

## 3. Add Your First Source

A **source** is a local directory or git repository containing agent assets organized by type.

```bash
# Local directory
nd source add ~/my-coding-assets

# Git repository (GitHub shorthand)
nd source add owner/repo

# Git repository (full URL)
nd source add https://github.com/owner/repo.git
```

nd scans the source for assets organized in convention-based directories (`skills/`, `agents/`, `commands/`, etc.). See [Creating Sources](creating-sources.md) for how to structure your own.

## 4. Browse Available Assets

List all assets discovered from your sources:

```bash
nd list
```

Filter by type:

```bash
nd list --type skills
```

Assets marked with `*` are already deployed.

## 5. Deploy an Asset

Deploy an asset by creating a symlink in your agent's config directory:

```bash
nd deploy skills/greeting
```

Deploy multiple assets at once:

```bash
nd deploy skills/greeting commands/hello agents/researcher
```

Or run `nd deploy` with no arguments to get an interactive picker.

## 6. Verify

Check that everything is healthy:

```bash
nd status
```

You should see your deployed assets with health indicators (checkmarks for healthy symlinks).

For a deeper health check of your entire setup:

```bash
nd doctor
```

## 7. Optional Setup

### Shell Completions

Enable tab-completion for your shell:

```bash
# Bash
nd completion bash --install

# Zsh
nd completion zsh --install

# Fish
nd completion fish --install
```

For zsh, you may need to add this to your `~/.zshrc` if not already present:

```bash
fpath+=~/.zfunc
autoload -Uz compinit && compinit
```

### Edit Configuration

Open your config file in your default editor:

```bash
nd settings edit
```

## Next Steps

- **[User Guide](user-guide.md)** -- Learn about managing sources, scopes, syncing, and more
- **[Profiles & Snapshots](profiles-and-snapshots.md)** -- Group assets into profiles and switch between them
- **[Configuration](configuration.md)** -- Customize nd behavior
- **[Creating Sources](creating-sources.md)** -- Build and share your own asset libraries
- **TUI Dashboard** -- Run `nd` with no arguments to launch the interactive dashboard
