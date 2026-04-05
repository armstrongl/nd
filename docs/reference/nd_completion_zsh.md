---
title: "nd completion zsh"
description: "Generate zsh completion script"
weight: 50
---

Generate zsh completion script

## Synopsis

Generate zsh completion script for nd.

To install completions:
  nd completion zsh --install

Or manually:
  nd completion zsh > ~/.zfunc/_nd

Then add to ~/.zshrc (if not already present):
  fpath+=~/.zfunc
  autoload -Uz compinit && compinit

```
nd completion zsh [flags]
```

## Examples

```
  # Print zsh completion script
  nd completion zsh

  # Auto-install to ~/.zfunc
  nd completion zsh --install

  # Install to custom directory
  nd completion zsh --install-dir ~/.my-completions
```

## Options

```
  -h, --help                 help for zsh
      --install              install to standard location
      --install-dir string   override install directory
```

## Options inherited from parent commands

```
      --config string   path to config file (default "~/.config/nd/config.yaml")
      --dry-run         show what would happen without making changes
      --json            output in JSON format
      --no-color        disable colored output
  -q, --quiet           suppress non-error output
  -s, --scope string    deployment scope (global|project) (default "global")
  -v, --verbose         verbose output to stderr
  -y, --yes             skip confirmation prompts
```

## Related

- [nd completion](nd_completion.md) - Generate shell completion scripts
