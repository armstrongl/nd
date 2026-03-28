---
title: "nd completion bash"
weight: 30
---


Generate bash completion script

## Synopsis

Generate bash completion script for nd.

To install completions:
  nd completion bash --install

Or manually:
  nd completion bash > ~/.local/share/bash-completion/completions/nd

Then restart your shell or source the file.

```
nd completion bash [flags]
```

## Options

```
  -h, --help                 help for bash
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

## SEE ALSO

* [nd completion](nd_completion.md)	 - Generate shell completion scripts

