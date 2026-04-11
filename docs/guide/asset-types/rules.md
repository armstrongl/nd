---
title: "Rules"
description: "Load when modifying rule file scanning, rule deployment, or the rules asset type."
lastValidated: "2026-04-04"
maxAgeDays: 90
weight: 50
paths:
  - "internal/sourcemanager/scanner.go"
  - "internal/deploy/**"
tags:
  - rules
  - assets
  - deployment
---

Use rules when you want to enforce a specific constraint or convention that the agent must follow in every interaction — such as "never use emojis" or "always write tests." Unlike context, which provides broad project knowledge, each rule targets a single, enforceable behavior.

Rules are a Claude Code-only asset type. Copilot CLI does not support rules.

Rules are assets that define behavioral constraints or conventions a coding agent must follow throughout a session. A rule can be a single Markdown file or a directory.

## Directory layout

```text
rules/
├── no-emojis.md
├── always-test.md
└── security-standards/
    └── ...
```

## File format

Each rule is a markdown file whose base filename describes the constraint it encodes. The file body states the rule in plain language. Frontmatter is optional.

## Deploy behavior

nd symlinks the individual file into the target location (see [How nd works](../how-nd-works.md) for details on the symlink strategy). Running [`nd deploy`](../../reference/nd_deploy.md) `rules/no-emojis` produces:

```text
~/.claude/rules/no-emojis.md → <source>/rules/no-emojis.md
```

## Scope rules

| Scope | Target path |
|-------|-------------|
| Global | `~/.claude/rules/<name>.md` |
| Project | `.claude/rules/<name>.md` |

To undeploy a rule, run [`nd remove`](../../reference/nd_remove.md) `rules/no-emojis`.

## Related

- [Asset type comparison](../creating-sources.md#asset-types) for a side-by-side overview of all types
- [Context](context.md) — provide broad project knowledge rather than individual constraints
- [Agents](agents.md) — define a full persona that bundles multiple behavioral instructions
- [Glossary: Rule](../glossary.md#rule) — terminology definition

## Create a rule

```shell
cat > ~/my-assets/rules/no-emojis.md << 'EOF'
Never use emojis in code comments, commit messages, or documentation unless the user explicitly requests them.
EOF
nd deploy rules/no-emojis
```
