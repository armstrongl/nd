package export_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/export"
)

// writeHooksJSON is a test helper that writes a hooks.json file to the given directory.
func writeHooksJSON(t *testing.T, dir string, content map[string]any) {
	t.Helper()
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal hooks.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), data, 0o644); err != nil {
		t.Fatalf("write hooks.json: %v", err)
	}
}

// makeHookDir creates a hook directory with the given hooks.json content and optional script files.
// Returns the absolute path to the directory.
func makeHookDir(t *testing.T, parent, name string, hooksContent map[string]any, scripts map[string]string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	writeHooksJSON(t, dir, hooksContent)
	for filename, content := range scripts {
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o755); err != nil {
			t.Fatalf("write script %s: %v", filename, err)
		}
	}
	return dir
}

func TestMergeHooks_SingleHookDir(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo hello"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "lint-hook", hooksContent, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "lint-hook", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The merged config should match the original
	hooks, ok := result.Config["hooks"].(map[string]any)
	if !ok {
		t.Fatal("merged config missing 'hooks' key")
	}
	preToolUse, ok := hooks["PreToolUse"]
	if !ok {
		t.Fatal("merged config missing 'PreToolUse' event")
	}
	arr, ok := preToolUse.([]any)
	if !ok {
		t.Fatal("PreToolUse is not an array")
	}
	if len(arr) != 1 {
		t.Fatalf("PreToolUse has %d matcher groups, want 1", len(arr))
	}

	// No scripts expected
	if len(result.Scripts) != 0 {
		t.Fatalf("expected no scripts, got %d", len(result.Scripts))
	}
}

func TestMergeHooks_TwoHooksDifferentEvents(t *testing.T) {
	tmp := t.TempDir()

	hook1Content := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo pre"},
					},
				},
			},
		},
	}
	hook2Content := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Write",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo post"},
					},
				},
			},
		},
	}

	dir1 := makeHookDir(t, tmp, "hook-a", hook1Content, nil)
	dir2 := makeHookDir(t, tmp, "hook-b", hook2Content, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "hook-a", Path: dir1},
		{Name: "hook-b", Path: dir2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := result.Config["hooks"].(map[string]any)

	if _, ok := hooks["PreToolUse"]; !ok {
		t.Fatal("missing PreToolUse event in merged config")
	}
	if _, ok := hooks["PostToolUse"]; !ok {
		t.Fatal("missing PostToolUse event in merged config")
	}
}

func TestMergeHooks_SameEventConcatenated(t *testing.T) {
	tmp := t.TempDir()

	hook1Content := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo first"},
					},
				},
			},
		},
	}
	hook2Content := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Write",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo second"},
					},
				},
			},
		},
	}

	dir1 := makeHookDir(t, tmp, "hook-a", hook1Content, nil)
	dir2 := makeHookDir(t, tmp, "hook-b", hook2Content, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "hook-a", Path: dir1},
		{Name: "hook-b", Path: dir2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := result.Config["hooks"].(map[string]any)
	postToolUse := hooks["PostToolUse"].([]any)

	if len(postToolUse) != 2 {
		t.Fatalf("PostToolUse has %d matcher groups, want 2", len(postToolUse))
	}

	// Verify order: first hook's matcher group first
	group0 := postToolUse[0].(map[string]any)
	if group0["matcher"] != "Bash" {
		t.Fatalf("first matcher group matcher = %q, want %q", group0["matcher"], "Bash")
	}
	group1 := postToolUse[1].(map[string]any)
	if group1["matcher"] != "Write" {
		t.Fatalf("second matcher group matcher = %q, want %q", group1["matcher"], "Write")
	}
}

func TestMergeHooks_SameEventSameMatcherNotDeduplicated(t *testing.T) {
	tmp := t.TempDir()

	// Both hooks define PostToolUse with the same "Bash" matcher
	hook1Content := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo first"},
					},
				},
			},
		},
	}
	hook2Content := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo second"},
					},
				},
			},
		},
	}

	dir1 := makeHookDir(t, tmp, "hook-a", hook1Content, nil)
	dir2 := makeHookDir(t, tmp, "hook-b", hook2Content, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "hook-a", Path: dir1},
		{Name: "hook-b", Path: dir2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := result.Config["hooks"].(map[string]any)
	postToolUse := hooks["PostToolUse"].([]any)

	// Both matcher groups should be kept even though they have the same matcher
	if len(postToolUse) != 2 {
		t.Fatalf("PostToolUse has %d matcher groups, want 2 (not deduplicated)", len(postToolUse))
	}
}

func TestMergeHooks_ScriptPathRewritten(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "lint.sh --strict"},
					},
				},
			},
		},
	}
	scripts := map[string]string{
		"lint.sh": "#!/bin/bash\necho lint",
	}
	dir := makeHookDir(t, tmp, "pre-commit-lint", hooksContent, scripts)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "pre-commit-lint", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The command should be rewritten to use ${CLAUDE_PLUGIN_ROOT}/scripts/<hook-name>/<filename>
	hooks := result.Config["hooks"].(map[string]any)
	preToolUse := hooks["PreToolUse"].([]any)
	group := preToolUse[0].(map[string]any)
	innerHooks := group["hooks"].([]any)
	handler := innerHooks[0].(map[string]any)
	cmd := handler["command"].(string)

	expectedCmd := "${CLAUDE_PLUGIN_ROOT}/scripts/pre-commit-lint/lint.sh --strict"
	if cmd != expectedCmd {
		t.Fatalf("command = %q, want %q", cmd, expectedCmd)
	}

	// Check that the script is in the Scripts map
	scriptSrc := filepath.Join(dir, "lint.sh")
	destPath, ok := result.Scripts[scriptSrc]
	if !ok {
		t.Fatalf("script %s not found in Scripts map; got keys: %v", scriptSrc, keysOf(result.Scripts))
	}
	expectedDest := "pre-commit-lint/lint.sh"
	if destPath != expectedDest {
		t.Fatalf("script dest = %q, want %q", destPath, expectedDest)
	}
}

func TestMergeHooks_InlineCommandUnchanged(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Write|Edit",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo done && exit 0"},
					},
				},
			},
		},
	}
	// No scripts in this hook directory
	dir := makeHookDir(t, tmp, "inline-hook", hooksContent, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "inline-hook", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Command should be left as-is
	hooks := result.Config["hooks"].(map[string]any)
	postToolUse := hooks["PostToolUse"].([]any)
	group := postToolUse[0].(map[string]any)
	innerHooks := group["hooks"].([]any)
	handler := innerHooks[0].(map[string]any)
	cmd := handler["command"].(string)

	if cmd != "echo done && exit 0" {
		t.Fatalf("inline command was modified: got %q", cmd)
	}

	// No scripts should be recorded
	if len(result.Scripts) != 0 {
		t.Fatalf("expected no scripts, got %d", len(result.Scripts))
	}
}

func TestMergeHooks_NoScriptsConfigOnly(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo session started"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "session-hook", hooksContent, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "session-hook", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Scripts) != 0 {
		t.Fatalf("expected empty Scripts map, got %d entries", len(result.Scripts))
	}

	hooks := result.Config["hooks"].(map[string]any)
	if _, ok := hooks["SessionStart"]; !ok {
		t.Fatal("missing SessionStart in merged config")
	}
}

func TestMergeHooks_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "bad-hook")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := export.MergeHooks([]export.HookDir{
		{Name: "bad-hook", Path: dir},
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "bad-hook") {
		t.Fatalf("error should name the hook directory: %v", err)
	}
}

func TestMergeHooks_MissingHooksKey(t *testing.T) {
	tmp := t.TempDir()

	// Valid JSON but no "hooks" key
	content := map[string]any{
		"notHooks": map[string]any{},
	}
	dir := makeHookDir(t, tmp, "no-hooks-key", content, nil)

	_, err := export.MergeHooks([]export.HookDir{
		{Name: "no-hooks-key", Path: dir},
	})
	if err == nil {
		t.Fatal("expected error for missing 'hooks' key")
	}
	if !strings.Contains(err.Error(), "no-hooks-key") {
		t.Fatalf("error should name the hook directory: %v", err)
	}
}

func TestMergeHooks_UnrecognizedEventName(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"InvalidEventName": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo test"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "bad-event-hook", hooksContent, nil)

	_, err := export.MergeHooks([]export.HookDir{
		{Name: "bad-event-hook", Path: dir},
	})
	if err == nil {
		t.Fatal("expected error for unrecognized event name")
	}
	if !strings.Contains(err.Error(), "bad-event-hook") {
		t.Fatalf("error should name the hook directory: %v", err)
	}
	if !strings.Contains(err.Error(), "InvalidEventName") {
		t.Fatalf("error should name the invalid event: %v", err)
	}
}

func TestMergeHooks_MultipleScriptsFromMultipleHooks(t *testing.T) {
	tmp := t.TempDir()

	hook1Content := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "lint.sh"},
					},
				},
			},
		},
	}
	hook2Content := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Write",
					"hooks": []any{
						map[string]any{"type": "command", "command": "format.sh"},
					},
				},
			},
		},
	}

	dir1 := makeHookDir(t, tmp, "lint-hook", hook1Content, map[string]string{
		"lint.sh": "#!/bin/bash\necho lint",
	})
	dir2 := makeHookDir(t, tmp, "format-hook", hook2Content, map[string]string{
		"format.sh": "#!/bin/bash\necho format",
	})

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "lint-hook", Path: dir1},
		{Name: "format-hook", Path: dir2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both scripts should be in the Scripts map with correct subdirectories
	if len(result.Scripts) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(result.Scripts))
	}

	lintSrc := filepath.Join(dir1, "lint.sh")
	if dest, ok := result.Scripts[lintSrc]; !ok || dest != "lint-hook/lint.sh" {
		t.Fatalf("lint.sh: got dest=%q ok=%v, want %q", dest, ok, "lint-hook/lint.sh")
	}

	formatSrc := filepath.Join(dir2, "format.sh")
	if dest, ok := result.Scripts[formatSrc]; !ok || dest != "format-hook/format.sh" {
		t.Fatalf("format.sh: got dest=%q ok=%v, want %q", dest, ok, "format-hook/format.sh")
	}
}

func TestMergeHooks_EmptyInput(t *testing.T) {
	result, err := export.MergeHooks(nil)
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	hooks, ok := result.Config["hooks"].(map[string]any)
	if !ok {
		t.Fatal("merged config missing 'hooks' key")
	}
	if len(hooks) != 0 {
		t.Fatalf("expected empty hooks, got %d events", len(hooks))
	}
	if len(result.Scripts) != 0 {
		t.Fatalf("expected no scripts, got %d", len(result.Scripts))
	}
}

func TestMergeHooks_NonCommandTypesNotRewritten(t *testing.T) {
	tmp := t.TempDir()

	// Hook with "http" and "prompt" types — these should not have command rewriting
	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "http", "url": "https://example.com/hook"},
						map[string]any{"type": "prompt", "prompt": "Check the output"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "non-cmd-hook", hooksContent, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "non-cmd-hook", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := result.Config["hooks"].(map[string]any)
	postToolUse := hooks["PostToolUse"].([]any)
	group := postToolUse[0].(map[string]any)
	innerHooks := group["hooks"].([]any)

	// http handler should be unchanged
	httpHandler := innerHooks[0].(map[string]any)
	if httpHandler["type"] != "http" {
		t.Fatalf("expected http type, got %v", httpHandler["type"])
	}
	if httpHandler["url"] != "https://example.com/hook" {
		t.Fatalf("http url changed: %v", httpHandler["url"])
	}

	// prompt handler should be unchanged
	promptHandler := innerHooks[1].(map[string]any)
	if promptHandler["type"] != "prompt" {
		t.Fatalf("expected prompt type, got %v", promptHandler["type"])
	}

	if len(result.Scripts) != 0 {
		t.Fatalf("expected no scripts, got %d", len(result.Scripts))
	}
}

func TestMergeHooks_ScriptWithAbsolutePath(t *testing.T) {
	tmp := t.TempDir()

	dir := filepath.Join(tmp, "abs-path-hook")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "check.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho check"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Command references the script by absolute path
	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": scriptPath},
					},
				},
			},
		},
	}
	writeHooksJSON(t, dir, hooksContent)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "abs-path-hook", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The absolute path should be rewritten
	hooks := result.Config["hooks"].(map[string]any)
	preToolUse := hooks["PreToolUse"].([]any)
	group := preToolUse[0].(map[string]any)
	innerHooks := group["hooks"].([]any)
	handler := innerHooks[0].(map[string]any)
	cmd := handler["command"].(string)

	expectedCmd := "${CLAUDE_PLUGIN_ROOT}/scripts/abs-path-hook/check.sh"
	if cmd != expectedCmd {
		t.Fatalf("command = %q, want %q", cmd, expectedCmd)
	}

	if _, ok := result.Scripts[scriptPath]; !ok {
		t.Fatalf("script %s not in Scripts map", scriptPath)
	}
}

func TestMergeHooks_ScriptWithArguments(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "lint.sh --strict --verbose"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "args-hook", hooksContent, map[string]string{
		"lint.sh": "#!/bin/bash\necho lint",
	})

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "args-hook", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := result.Config["hooks"].(map[string]any)
	preToolUse := hooks["PreToolUse"].([]any)
	group := preToolUse[0].(map[string]any)
	innerHooks := group["hooks"].([]any)
	handler := innerHooks[0].(map[string]any)
	cmd := handler["command"].(string)

	// The script name should be rewritten, arguments preserved
	expectedCmd := "${CLAUDE_PLUGIN_ROOT}/scripts/args-hook/lint.sh --strict --verbose"
	if cmd != expectedCmd {
		t.Fatalf("command = %q, want %q", cmd, expectedCmd)
	}
}

// --- ValidateHooksJSON tests ---

func TestValidateHooksJSON_Valid(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "Bash",
					"hooks": [
						{"type": "command", "command": "echo hello"}
					]
				}
			]
		}
	}`)
	if err := export.ValidateHooksJSON(data); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHooksJSON_AllEventNames(t *testing.T) {
	// Test that all recognized event names are accepted
	events := []string{
		"PreToolUse", "PostToolUse", "PostToolUseFailure",
		"UserPromptSubmit", "SessionStart", "SessionEnd",
		"Stop", "Notification", "SubagentStart", "SubagentStop",
		"PermissionRequest", "PreCompact", "PostCompact",
		"WorktreeCreate", "WorktreeRemove", "Elicitation",
		"ElicitationResult", "ConfigChange", "InstructionsLoaded",
		"TaskCompleted", "TeammateIdle", "StopFailure",
	}
	for _, event := range events {
		t.Run(event, func(t *testing.T) {
			config := map[string]any{
				"hooks": map[string]any{
					event: []any{
						map[string]any{
							"matcher": "",
							"hooks": []any{
								map[string]any{"type": "command", "command": "echo test"},
							},
						},
					},
				},
			}
			data, _ := json.Marshal(config)
			if err := export.ValidateHooksJSON(data); err != nil {
				t.Fatalf("event %q should be valid: %v", event, err)
			}
		})
	}
}

func TestValidateHooksJSON_AllHandlerTypes(t *testing.T) {
	types := []string{"command", "http", "prompt", "agent"}
	for _, handlerType := range types {
		t.Run(handlerType, func(t *testing.T) {
			config := map[string]any{
				"hooks": map[string]any{
					"PreToolUse": []any{
						map[string]any{
							"matcher": "",
							"hooks": []any{
								map[string]any{"type": handlerType},
							},
						},
					},
				},
			}
			data, _ := json.Marshal(config)
			if err := export.ValidateHooksJSON(data); err != nil {
				t.Fatalf("handler type %q should be valid: %v", handlerType, err)
			}
		})
	}
}

func TestValidateHooksJSON_MalformedJSON(t *testing.T) {
	err := export.ValidateHooksJSON([]byte("{not valid json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestValidateHooksJSON_MissingHooksKey(t *testing.T) {
	data := []byte(`{"notHooks": {}}`)
	err := export.ValidateHooksJSON(data)
	if err == nil {
		t.Fatal("expected error for missing 'hooks' key")
	}
	if !strings.Contains(err.Error(), "hooks") {
		t.Fatalf("error should mention 'hooks': %v", err)
	}
}

func TestValidateHooksJSON_UnrecognizedEvent(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"BogusEvent": [
				{
					"matcher": "",
					"hooks": [{"type": "command", "command": "echo test"}]
				}
			]
		}
	}`)
	err := export.ValidateHooksJSON(data)
	if err == nil {
		t.Fatal("expected error for unrecognized event name")
	}
	if !strings.Contains(err.Error(), "BogusEvent") {
		t.Fatalf("error should name the bad event: %v", err)
	}
}

func TestValidateHooksJSON_InvalidHandlerType(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "",
					"hooks": [{"type": "websocket", "command": "echo test"}]
				}
			]
		}
	}`)
	err := export.ValidateHooksJSON(data)
	if err == nil {
		t.Fatal("expected error for invalid handler type")
	}
	if !strings.Contains(err.Error(), "websocket") {
		t.Fatalf("error should name the bad type: %v", err)
	}
}

func TestValidateHooksJSON_MissingHandlerType(t *testing.T) {
	data := []byte(`{
		"hooks": {
			"PreToolUse": [
				{
					"matcher": "",
					"hooks": [{"command": "echo test"}]
				}
			]
		}
	}`)
	err := export.ValidateHooksJSON(data)
	if err == nil {
		t.Fatal("expected error for missing handler type")
	}
}

func TestValidateHooksJSON_EmptyHooksObject(t *testing.T) {
	data := []byte(`{"hooks": {}}`)
	// An empty hooks object is structurally valid
	if err := export.ValidateHooksJSON(data); err != nil {
		t.Fatalf("empty hooks object should be valid: %v", err)
	}
}

func TestValidateHooksJSON_HooksNotObject(t *testing.T) {
	data := []byte(`{"hooks": "not an object"}`)
	err := export.ValidateHooksJSON(data)
	if err == nil {
		t.Fatal("expected error when hooks is not an object")
	}
}

func TestMergeHooks_MissingHooksJSONFile(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "empty-hook")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// No hooks.json file

	_, err := export.MergeHooks([]export.HookDir{
		{Name: "empty-hook", Path: dir},
	})
	if err == nil {
		t.Fatal("expected error for missing hooks.json")
	}
	if !strings.Contains(err.Error(), "empty-hook") {
		t.Fatalf("error should name the hook directory: %v", err)
	}
}

func TestMergeHooks_InvalidHandlerTypeInMerge(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "invalidtype", "command": "echo test"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "bad-type-hook", hooksContent, nil)

	_, err := export.MergeHooks([]export.HookDir{
		{Name: "bad-type-hook", Path: dir},
	})
	if err == nil {
		t.Fatal("expected error for invalid handler type")
	}
	if !strings.Contains(err.Error(), "bad-type-hook") {
		t.Fatalf("error should name the hook directory: %v", err)
	}
}

func TestMergeHooks_MultipleEventsFromSingleHook(t *testing.T) {
	tmp := t.TempDir()

	hooksContent := map[string]any{
		"hooks": map[string]any{
			"PreToolUse": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo pre"},
					},
				},
			},
			"PostToolUse": []any{
				map[string]any{
					"matcher": "Write",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo post"},
					},
				},
			},
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks": []any{
						map[string]any{"type": "command", "command": "echo start"},
					},
				},
			},
		},
	}
	dir := makeHookDir(t, tmp, "multi-event", hooksContent, nil)

	result, err := export.MergeHooks([]export.HookDir{
		{Name: "multi-event", Path: dir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hooks := result.Config["hooks"].(map[string]any)
	if len(hooks) != 3 {
		t.Fatalf("expected 3 events, got %d", len(hooks))
	}
	for _, event := range []string{"PreToolUse", "PostToolUse", "SessionStart"} {
		if _, ok := hooks[event]; !ok {
			t.Fatalf("missing event %s", event)
		}
	}
}

// keysOf returns the keys of a map for diagnostic output.
func keysOf(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
