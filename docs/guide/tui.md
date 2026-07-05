---
title: "Interactive TUI"
description: "Load when modifying the interactive terminal UI: screens, navigation, key bindings, header, or scope switching."
lastValidated: "2026-07-05"
maxAgeDays: 90
weight: 35
paths:
  - "internal/tui/**"
  - "cmd/root.go"
tags:
  - tui
  - interactive
  - navigation
---

Run `nd` with no arguments in a terminal to open the interactive TUI: a menu-driven interface for deploying, removing, and managing assets without memorizing flags. Every action available in the TUI has a command-line equivalent, so use whichever fits the moment.

## Launch the TUI

Run the bare command:

```shell {filename="Terminal"}
nd
```

nd opens the TUI when two conditions hold:

- Standard input is a terminal (a TTY). In a pipe, script, or CI job, nd prints help instead.
- None of `--verbose`, `--quiet`, or `--json` is set. Those flags signal non-interactive use, so nd prints help rather than opening the TUI.

When either condition fails, `nd` with no subcommand prints the top-level help text. To open the interface explicitly while those flags are set, drop the conflicting flag.

## The layout

The TUI draws three regions stacked vertically: a header, the active screen, and a help bar.

```text {filename="TUI layout"}
  no profile · global · claude-code                 3 deployed  0 issues   ← header
                                                                            ← blank
  Deploy assets                                                             ← active screen
  Remove assets
  Browse assets
  ...
                                                                            ← blank
  esc back  j/k navigate  enter select  ? help  q quit                      ← help bar
```

- **Header:** shows the active profile, the current scope (`global` or `project`), and the target agent, separated by dots. When `--dry-run` is active, a `[DRY RUN]` prefix appears. The right side shows the deployed asset count and the number with health issues (highlighted when above zero).
- **Active screen:** the current step, such as the main menu or an asset picker.
- **Help bar:** the key bindings available on the current screen. The bar updates as you move between screens and steps.

The header re-queries live state, so the scope, agent, and counts stay current as you deploy, remove, or switch scope.

## Navigate with the keyboard

These keys work on every screen unless a text input is focused:

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up through a list |
| `enter` | Select the highlighted item |
| `esc` | Go back one screen (quits from the top-level menu) |
| `ctrl+s` | Switch scope between global and project |
| `q` | Quit the TUI |
| `ctrl+c` | Quit immediately |

When a text input is focused (for example, a filter box or the export name field), typing goes to the field and the global keys are suppressed. Only `ctrl+c` still force-quits. Press `enter` or `esc` to leave the field, then the global keys apply again.

Some screens add their own keys, shown in the help bar:

- **Multi-select pickers** (deploy, remove): `x` or `space` toggles an item, `enter` confirms the selection.
- **Browse and status:** `/` opens a filter box; type to narrow the list, `enter` applies, `esc` cancels.
- **Status:** `d` deploys, `r` removes, and `f` fixes the highlighted asset in place.
- **Confirmation prompts:** `h` / `l` move between yes and no, `enter` confirms.

## The main menu

The main menu groups actions under three headers. The separator rows (`── Manage ──`, `── System ──`) are labels, not selectable options.

| Group | Option | Opens |
|-------|--------|-------|
| Actions | Deploy assets | Pick a type, then select assets to deploy |
| Actions | Remove assets | Select deployed assets to remove |
| Actions | Browse assets | Scroll and filter every discovered asset |
| Actions | View status | Show deployed assets and health |
| Actions | Run doctor | Run the health checks |
| Manage | Switch profile | Pick and activate a profile |
| Manage | Manage snapshots | Save, restore, or delete snapshots |
| Manage | Pin/Unpin assets | Pin assets so profile switches keep them |
| Manage | Manage sources | Add, list, sync, or remove sources |
| Manage | Export plugin | Package assets (see note below) |
| System | Switch scope | Choose global or project scope |
| System | Settings | Open the config file in your editor |
| System | Quit | Exit the TUI |

Plugin export runs from the command line with [`nd export`](../reference/nd_export.md); the menu entry is a placeholder that returns to the menu. See [Export a plugin or marketplace](export-plugin-workflow.md) for the full workflow.

## Screens

Each menu action pushes a screen onto a stack. Press `esc` to pop back to the previous screen, or keep pressing `esc` to return to the main menu and then quit.

- **Deploy:** pick an asset type, multi-select assets, and confirm. Conflicts prompt for a yes/no decision before overwriting.
- **Remove:** multi-select deployed assets and confirm. Pinned assets prompt for explicit confirmation.
- **Browse:** a scrollable, filterable list of every discovered asset, with a deploy shortcut.
- **Status:** the deployed assets with health indicators, plus inline deploy, remove, and fix shortcuts.
- **Doctor:** the same five health checks as [`nd doctor`](../reference/nd_doctor.md).
- **Profile:** switch the active profile.
- **Snapshot:** save, restore, or delete snapshots.
- **Pin:** pin or unpin assets.
- **Source:** add, list, sync, or remove sources.
- **Settings:** open the config file in your editor.

## Switch scope

Scope decides where assets deploy: the global agent directory (`~/.claude/`) or the project directory (`.claude/`). Switch it two ways:

- Press `ctrl+s` on any screen to toggle between global and project.
- Choose **Switch scope** from the main menu and pick a scope.

Project scope needs a project root. If nd cannot detect one (you are not inside a project directory), the toggle does nothing and the menu form reports that no project root was detected. The header reflects the active scope as soon as it changes.

## First run

The first time you open the TUI without any user-defined sources, nd shows a first-run screen that walks you through adding your first source instead of the main menu. Once at least one source is registered, the main menu opens by default. See [Create asset sources](creating-sources.md) for how sources are structured.

## When the TUI does not open

If `nd` prints help instead of opening the interface, check for:

- A non-terminal standard input (running inside a pipe, a script, or CI).
- A `--verbose`, `--quiet`, or `--json` flag on the command line.

Run individual commands directly in those environments. Every TUI action maps to a command, such as [`nd deploy`](../reference/nd_deploy.md), [`nd status`](../reference/nd_status.md), or [`nd profile switch`](../reference/nd_profile_switch.md).

## Next steps

- **[User guide](user-guide.md):** The command-line equivalents for every TUI action
- **[How nd works](how-nd-works.md):** What happens on disk when you deploy from any interface
- **[Profiles and snapshots](profiles-and-snapshots.md):** Group assets into profiles and switch between them
- **[Export a plugin or marketplace](export-plugin-workflow.md):** Package assets for distribution
- **[Troubleshooting](troubleshooting.md):** Fix broken symlinks, missing assets, and other common issues
