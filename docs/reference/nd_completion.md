---
title: "nd completion"
description: "Generate shell completion scripts"
weight: 20
---

Generate shell completion scripts

## Synopsis

Generate shell completion scripts for nd.

Available shells: bash, zsh, fish

Run "nd completion <shell> --help" for shell-specific instructions.

## Examples

```
  nd completion bash
  nd completion zsh --install
  nd completion fish
```

## Options

```
  -h, --help   help for completion
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

- [nd](nd.md) - Napoleon Dynamite - coding agent asset manager
- [nd completion bash](nd_completion_bash.md) - Generate bash completion script
- [nd completion fish](nd_completion_fish.md) - Generate fish completion script
- [nd completion zsh](nd_completion_zsh.md) - Generate zsh completion script
