package nd

// SymlinkStrategy controls how symlinks are created.
type SymlinkStrategy string

const (
	SymlinkAbsolute SymlinkStrategy = "absolute"
	SymlinkRelative SymlinkStrategy = "relative"
)
