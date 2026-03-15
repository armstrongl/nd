package deploy

import "fmt"

// Action describes what a deploy/remove operation did.
type Action int

const (
	ActionCreated  Action = iota // New symlink created
	ActionRemoved                // Symlink removed
	ActionReplaced               // Existing symlink replaced
	ActionSkipped                // No action needed (already correct)
	ActionBackedUp               // Existing file backed up before replace (context files)
	ActionFailed                 // Operation failed
	ActionDryRun                 // Would have done this (dry-run mode)
)

// String returns the human-readable name of the action.
func (a Action) String() string {
	switch a {
	case ActionCreated:
		return "created"
	case ActionRemoved:
		return "removed"
	case ActionReplaced:
		return "replaced"
	case ActionSkipped:
		return "skipped"
	case ActionBackedUp:
		return "backed-up"
	case ActionFailed:
		return "failed"
	case ActionDryRun:
		return "dry-run"
	default:
		return "unknown"
	}
}

// MarshalJSON implements json.Marshaler for Action.
func (a Action) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", a.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler for Action.
func (a *Action) UnmarshalJSON(data []byte) error {
	var s string
	if len(data) < 2 || data[0] != '"' {
		return fmt.Errorf("invalid action: %s", data)
	}
	s = string(data[1 : len(data)-1])
	switch s {
	case "created":
		*a = ActionCreated
	case "removed":
		*a = ActionRemoved
	case "replaced":
		*a = ActionReplaced
	case "skipped":
		*a = ActionSkipped
	case "backed-up":
		*a = ActionBackedUp
	case "failed":
		*a = ActionFailed
	case "dry-run":
		*a = ActionDryRun
	default:
		return fmt.Errorf("unknown action: %q", s)
	}
	return nil
}
