package builtin

import "embed"

// FS holds the embedded built-in source directory tree.
// The "source" subdirectory contains standard nd asset layout:
// source/skills/, source/commands/, source/agents/.
//
//go:embed source
var FS embed.FS
