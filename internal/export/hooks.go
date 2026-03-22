package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HookDir identifies a hook directory to include in a merge operation.
type HookDir struct {
	Name string // hook directory name (e.g., "pre-commit-lint")
	Path string // absolute path to hook directory
}

// MergedHooks is the result of merging one or more hook directories.
type MergedHooks struct {
	Config  map[string]any    // merged {"hooks": {...}} content
	Scripts map[string]string // src absolute path -> dest path relative to scripts/ (e.g., "pre-commit-lint/lint.sh")
}

// validEventNames is the set of recognized Claude Code hook event names.
var validEventNames = map[string]bool{
	"PreToolUse":         true,
	"PostToolUse":        true,
	"PostToolUseFailure": true,
	"UserPromptSubmit":   true,
	"SessionStart":       true,
	"SessionEnd":         true,
	"Stop":               true,
	"Notification":       true,
	"SubagentStart":      true,
	"SubagentStop":       true,
	"PermissionRequest":  true,
	"PreCompact":         true,
	"PostCompact":        true,
	"WorktreeCreate":     true,
	"WorktreeRemove":     true,
	"Elicitation":        true,
	"ElicitationResult":  true,
	"ConfigChange":       true,
	"InstructionsLoaded": true,
	"TaskCompleted":      true,
	"TeammateIdle":       true,
	"StopFailure":        true,
}

// validHandlerTypes is the set of recognized hook handler types.
var validHandlerTypes = map[string]bool{
	"command": true,
	"http":    true,
	"prompt":  true,
	"agent":   true,
}

// MergeHooks reads hook directories and produces a merged hooks.json config
// plus a map of scripts to copy, organized by hook name subdirectory.
//
// For each hook dir, it reads and validates hooks.json, collects non-hooks.json
// files as potential scripts, then merges all event handler arrays by
// concatenation in the order the hook directories are provided.
//
// Command values that reference a file in the hook directory (by basename or
// absolute path) are rewritten to ${CLAUDE_PLUGIN_ROOT}/scripts/<hook-name>/<basename>
// and recorded in the Scripts map. Inline commands are left unchanged.
func MergeHooks(hookDirs []HookDir) (*MergedHooks, error) {
	merged := &MergedHooks{
		Config:  map[string]any{"hooks": map[string]any{}},
		Scripts: make(map[string]string),
	}

	if len(hookDirs) == 0 {
		return merged, nil
	}

	mergedHooks, ok := merged.Config["hooks"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("internal error: merged hooks config has unexpected type")
	}

	for _, hd := range hookDirs {
		// Read hooks.json
		hooksPath := filepath.Join(hd.Path, "hooks.json")
		data, err := os.ReadFile(hooksPath)
		if err != nil {
			return nil, fmt.Errorf("hook %q: %w", hd.Name, err)
		}

		// Validate the hooks.json
		if err := ValidateHooksJSON(data); err != nil {
			return nil, fmt.Errorf("hook %q: %w", hd.Name, err)
		}

		// Parse the validated JSON
		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("hook %q: %w", hd.Name, err)
		}

		// Collect script files (all non-hooks.json files in the directory)
		scriptFiles := collectScriptFiles(hd.Path)

		// Get the hooks object
		hooksObj, ok := parsed["hooks"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("hook %q: \"hooks\" is not an object", hd.Name)
		}

		// Process each event
		for eventName, matcherGroups := range hooksObj {
			groups, ok := matcherGroups.([]any)
			if !ok {
				return nil, fmt.Errorf("hook %q: event %q: expected array of matcher groups", hd.Name, eventName)
			}

			// Rewrite script references in command values
			for _, group := range groups {
				groupMap, ok := group.(map[string]any)
				if !ok {
					continue
				}
				innerHooks, ok := groupMap["hooks"].([]any)
				if !ok {
					continue
				}
				for _, handler := range innerHooks {
					handlerMap, ok := handler.(map[string]any)
					if !ok {
						continue
					}
					if handlerMap["type"] != "command" {
						continue
					}
					cmdVal, ok := handlerMap["command"].(string)
					if !ok {
						continue
					}
					rewritten, scriptSrc := rewriteCommand(cmdVal, hd.Name, hd.Path, scriptFiles)
					handlerMap["command"] = rewritten
					if scriptSrc != "" {
						merged.Scripts[scriptSrc] = filepath.Join(hd.Name, filepath.Base(scriptSrc))
					}
				}
			}

			// Concatenate matcher groups into the merged hooks
			existing, ok := mergedHooks[eventName]
			if ok {
				existingArr, ok := existing.([]any)
				if !ok {
					return nil, fmt.Errorf("hook %q: event %q: existing merged value has unexpected type", hd.Name, eventName)
				}
				mergedHooks[eventName] = append(existingArr, groups...)
			} else {
				// Make a copy of the slice to avoid aliasing
				copied := make([]any, len(groups))
				copy(copied, groups)
				mergedHooks[eventName] = copied
			}
		}
	}

	return merged, nil
}

// ValidateHooksJSON checks that a hooks.json is structurally valid:
// valid JSON, has "hooks" key as an object, recognized event names, valid hook types.
func ValidateHooksJSON(data []byte) error {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	hooksVal, ok := raw["hooks"]
	if !ok {
		return fmt.Errorf("missing required \"hooks\" key")
	}

	hooksObj, ok := hooksVal.(map[string]any)
	if !ok {
		return fmt.Errorf("\"hooks\" must be an object, got %T", hooksVal)
	}

	for eventName, matcherGroups := range hooksObj {
		if !validEventNames[eventName] {
			return fmt.Errorf("unrecognized event name %q", eventName)
		}

		groups, ok := matcherGroups.([]any)
		if !ok {
			return fmt.Errorf("event %q: expected array of matcher groups, got %T", eventName, matcherGroups)
		}

		for i, group := range groups {
			groupMap, ok := group.(map[string]any)
			if !ok {
				return fmt.Errorf("event %q: matcher group %d: expected object", eventName, i)
			}

			innerHooks, ok := groupMap["hooks"]
			if !ok {
				continue
			}
			innerArr, ok := innerHooks.([]any)
			if !ok {
				return fmt.Errorf("event %q: matcher group %d: \"hooks\" must be an array", eventName, i)
			}

			for j, handler := range innerArr {
				handlerMap, ok := handler.(map[string]any)
				if !ok {
					return fmt.Errorf("event %q: matcher group %d: handler %d: expected object", eventName, i, j)
				}

				typeVal, ok := handlerMap["type"]
				if !ok {
					return fmt.Errorf("event %q: matcher group %d: handler %d: missing \"type\" field", eventName, i, j)
				}

				typeStr, ok := typeVal.(string)
				if !ok {
					return fmt.Errorf("event %q: matcher group %d: handler %d: \"type\" must be a string", eventName, i, j)
				}

				if !validHandlerTypes[typeStr] {
					return fmt.Errorf("event %q: matcher group %d: handler %d: unrecognized type %q", eventName, i, j, typeStr)
				}
			}
		}
	}

	return nil
}

// collectScriptFiles returns a map of filename -> absolute path for all
// non-hooks.json files in the given directory (non-recursive).
func collectScriptFiles(dirPath string) map[string]string {
	files := make(map[string]string)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return files
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if entry.Name() == "hooks.json" {
			continue
		}
		// Skip symlinks to prevent following links outside the hook directory.
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		files[entry.Name()] = filepath.Join(dirPath, entry.Name())
	}
	return files
}

// rewriteCommand checks if a command value references a script file in the hook
// directory. If so, it rewrites the command to use ${CLAUDE_PLUGIN_ROOT}/scripts/<hookName>/<basename>
// and returns the absolute path to the source script. If the command is an inline
// command (no matching file), it returns the original command and an empty scriptSrc.
func rewriteCommand(cmd, hookName, hookPath string, scriptFiles map[string]string) (rewritten string, scriptSrc string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return cmd, ""
	}

	// Extract the first token (potential script path) from the command
	// Arguments follow after the first space
	parts := strings.SplitN(cmd, " ", 2)
	firstToken := parts[0]
	args := ""
	if len(parts) > 1 {
		args = " " + parts[1]
	}

	// Check if the first token is a basename that matches a file in the hook directory
	basename := filepath.Base(firstToken)
	if absPath, ok := scriptFiles[basename]; ok {
		rewrittenCmd := "${CLAUDE_PLUGIN_ROOT}/scripts/" + hookName + "/" + basename + args
		return rewrittenCmd, absPath
	}

	// Check if the first token is an absolute path to a file in the hook directory
	if filepath.IsAbs(firstToken) {
		tokenBase := filepath.Base(firstToken)
		if absPath, ok := scriptFiles[tokenBase]; ok {
			rewrittenCmd := "${CLAUDE_PLUGIN_ROOT}/scripts/" + hookName + "/" + tokenBase + args
			return rewrittenCmd, absPath
		}
	}

	// No match — inline command, leave as-is
	return cmd, ""
}
