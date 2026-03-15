package nd

// OriginalFileKind describes what kind of file already exists at a target path.
// Used by backup and conflict detection to determine warning severity.
type OriginalFileKind string

const (
	FileKindManagedSymlink OriginalFileKind = "nd-managed-symlink"
	FileKindForeignSymlink OriginalFileKind = "foreign-symlink"
	FileKindPlainFile      OriginalFileKind = "plain-file"
)
