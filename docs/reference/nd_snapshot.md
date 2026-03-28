## nd snapshot

Manage deployment snapshots

### Synopsis

Save, restore, list, and delete point-in-time deployment snapshots.

### Options

```
  -h, --help   help for snapshot
```

### Options inherited from parent commands

```
      --config string   path to config file (default "/Users/larah/.config/nd/config.yaml")
      --dry-run         show what would happen without making changes
      --json            output in JSON format
      --no-color        disable colored output
  -q, --quiet           suppress non-error output
  -s, --scope string    deployment scope (global|project) (default "global")
  -v, --verbose         verbose output to stderr
  -y, --yes             skip confirmation prompts
```

### SEE ALSO

- [nd](nd.md) - Napoleon Dynamite — coding agent asset manager
- [nd snapshot delete](nd_snapshot_delete.md) - Delete a snapshot
- [nd snapshot list](nd_snapshot_list.md) - List all snapshots
- [nd snapshot restore](nd_snapshot_restore.md) - Restore deployments from a snapshot
- [nd snapshot save](nd_snapshot_save.md) - Save current deployments as a named snapshot
