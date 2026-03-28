---
title: About
weight: -1
---

**nd** (Napoleon Dynamite) is a CLI tool for managing coding agent assets — skills, agents, commands, output styles, rules, context files, plugins, and hooks — via symlink deployment.

## How it works

nd creates symlinks from your agent's config directory to asset sources (local directories or git repositories). Edit the source, and the change shows up instantly in the deployed location. No redeploy needed.

## Links

- [GitHub repository](https://github.com/armstrongl/nd)
- [Releases](https://github.com/armstrongl/nd/releases)
- [Go package documentation](https://pkg.go.dev/github.com/armstrongl/nd)
- [Issue tracker](https://github.com/armstrongl/nd/issues)

## Install

```shell
# Homebrew (macOS/Linux)
brew install --cask armstrongl/tap/nd

# Go install
go install github.com/armstrongl/nd@latest
```

## License

nd is licensed under the [MIT License](https://github.com/armstrongl/nd/blob/main/LICENSE).
