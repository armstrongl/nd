# /nd-quickstart

Guide a new user through setting up nd and deploying their first assets. This command performs each step in sequence, checking current state before proceeding.

## Step 1: check initialization status

Run `nd status` to determine whether nd is already initialized.

- If nd returns output without error, it is initialized. Print "nd is already initialized." and skip to Step 3.
- If the command fails or reports that nd is not initialized, proceed to Step 2.

## Step 2: initialize nd

Run:

```shell
nd init
```

Confirm it succeeded by checking for the config file at `~/.config/nd/config.yaml`. If initialization failed, print the error output and stop. Do not proceed with a broken setup.

## Step 3: show available sources

Run:

```shell
nd source list
```

Display the results. Explain what each source is:

- **builtin**: Ships with nd, provides nd-specific skills, commands, and agents
- **local**: A directory on disk containing assets
- **git**: A cloned git repository containing assets

If the only source is `builtin`, suggest adding a source:

```
You have no custom sources yet. Add one with:
  nd source add ~/path/to/your/assets    (local directory)
  nd source add owner/repo               (GitHub repository)
```

## Step 4: browse available assets

Run:

```shell
nd list
```

Display the full asset list. Explain the output format:

- The type column shows the asset category (skills, commands, agents, rules, etc.)
- The source column shows which source provides the asset
- Assets marked with `*` are already deployed
- The name column is what you use in deploy commands

## Step 5: deploy a starter set

If there are undeployed assets available, offer to deploy them. Present the user with a choice:

1. **Deploy all available assets** -- Run `nd deploy <all-asset-refs>` with every undeployed asset
2. **Deploy by type** -- Ask which types to deploy, then deploy all assets of those types
3. **Pick individual assets** -- Run `nd deploy` with no arguments to launch the interactive picker
4. **Skip** -- Deploy nothing for now

Execute the chosen option. After deploying, run `nd status` and display the results.

## Step 6: provide next steps

Print the following guidance, adapting based on what was deployed:

```
Next steps:

  Manage assets:
    nd list                          Browse all available assets
    nd deploy <type/name>            Deploy an asset
    nd remove <type/name>            Remove a deployed asset
    nd status                        Check deployment health

  Add more sources:
    nd source add ~/my-assets        Add a local directory
    nd source add owner/repo         Add a GitHub repository
    nd sync --source <id>            Update a git source

  Organize with profiles:
    nd profile create work --from-current    Save current setup as a profile
    nd profile switch personal               Switch between profiles
    nd pin <type/name>                       Keep an asset across profile switches

  Get help:
    nd doctor                        Run a health check
    nd --help                        Show all commands
```

## Rules

- Run each nd command using the Bash tool, not by simulating output.
- Do not skip steps. Execute them in order, adapting based on the results of each step.
- If any command fails, display the error and suggest a remediation before continuing.
- Do not deploy assets without the user's confirmation.
- Use the exact nd CLI commands documented here. Do not invent flags or subcommands.
