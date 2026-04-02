---
title: "nd completion fish"
weight: 40
---

Generate fish completion script

## Synopsis

Generate fish completion script for nd.

To install completions:
  nd completion fish --install

Or manually:
  nd completion fish > ~/.config/fish/completions/nd.fish

```
nd completion fish [flags]
```

## Options

```
  -h, --help                 help for fish
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
