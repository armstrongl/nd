package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
)

// builtinAssets enumerates the six first-party assets embedded in the nd
// binary, keyed by their scan name (as reported by `nd list`) and the
// deploy reference used to select them.
var builtinAssets = []struct {
	Name    string // name as it appears in `nd list`
	Type    string // asset type dir
	Ref     string // `nd deploy` reference
	LinkRel string // expected symlink path relative to the agent global dir
}{
	{"nd-create-source", "skills", "skills/nd-create-source", "skills/nd-create-source"},
	{"nd-scaffold-asset", "skills", "skills/nd-scaffold-asset", "skills/nd-scaffold-asset"},
	{"nd-deploy-workflow", "skills", "skills/nd-deploy-workflow", "skills/nd-deploy-workflow"},
	{"nd-quickstart", "commands", "commands/nd-quickstart.md", "commands/nd-quickstart.md"},
	{"nd-audit", "commands", "commands/nd-audit.md", "commands/nd-audit.md"},
	{"nd-expert", "agents", "agents/nd-expert.md", "agents/nd-expert.md"},
}

// builtinEnv is a fully isolated environment for exercising the embedded
// builtin source through the compiled binary. Every path lives under a single
// t.TempDir() so the tests never touch the developer's real HOME, XDG cache,
// or agent config.
type builtinEnv struct {
	configPath string   // --config target (isolated)
	agentDir   string   // isolated claude-code global dir (deploy target)
	env        []string // exec environment (isolated HOME/XDG_CACHE_HOME/PATH)
}

// setupBuiltinEnv writes a valid, isolated config whose claude-code agent
// points at a temp global dir, and returns the environment the compiled
// binary must run under. The builtin source is injected automatically by the
// binary; it does not need to be listed in the config.
func setupBuiltinEnv(t *testing.T) builtinEnv {
	t.Helper()
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	cacheDir := filepath.Join(tmp, "cache")
	agentDir := filepath.Join(tmp, "agent")
	// binDir is intentionally empty so the subprocess PATH never resolves a
	// real `claude` binary; claude-code detection relies solely on agentDir.
	binDir := filepath.Join(tmp, "bin")
	configDir := filepath.Join(home, ".config", "nd")
	for _, d := range []string{cacheDir, agentDir, binDir, filepath.Join(configDir, "state")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	configPath := filepath.Join(configDir, "config.yaml")
	cfg := strings.Join([]string{
		"version: 1",
		"default_scope: global",
		"default_agent: claude-code",
		"symlink_strategy: absolute",
		"sources: []",
		"agents:",
		"  - name: claude-code",
		"    global_dir: " + agentDir,
	}, "\n") + "\n"
	if err := os.WriteFile(configPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return builtinEnv{
		configPath: configPath,
		agentDir:   agentDir,
		env: []string{
			"PATH=" + binDir,
			"HOME=" + home,
			"XDG_CACHE_HOME=" + cacheDir,
			"NO_COLOR=1",
		},
	}
}

// runNDEnv runs the compiled binary with an explicit (isolated) environment.
// It mirrors runND but sets cmd.Env so the builtin cache and agent dir resolve
// under the test's temp directory instead of the developer's real HOME.
func runNDEnv(t *testing.T, env []string, args ...string) runResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run nd %v: %v", args, err)
		}
	}
	return runResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// TestBuiltinSource_ListShowsBuiltinAssets verifies the embedded builtin
// source is discovered and its six assets are available via `nd list` (both
// human and JSON output) when driven through the compiled binary.
func TestBuiltinSource_ListShowsBuiltinAssets(t *testing.T) {
	env := setupBuiltinEnv(t)

	// Human output
	res := runNDEnv(t, env.env, "--config", env.configPath, "list")
	if res.ExitCode != 0 {
		t.Fatalf("nd list exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	for _, a := range builtinAssets {
		if !strings.Contains(res.Stdout, a.Name) {
			t.Errorf("expected %q in `nd list` output, got:\n%s", a.Name, res.Stdout)
		}
	}

	// JSON output — every asset must report source "builtin".
	jres := runNDEnv(t, env.env, "--config", env.configPath, "--json", "list")
	if jres.ExitCode != 0 {
		t.Fatalf("nd --json list exit %d, stderr: %s", jres.ExitCode, jres.Stderr)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   []struct {
			Type   string `json:"type"`
			Name   string `json:"name"`
			Source string `json:"source"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jres.Stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, jres.Stdout)
	}
	if envelope.Status != "ok" {
		t.Errorf("expected status ok, got %q", envelope.Status)
	}
	seen := make(map[string]bool)
	for _, e := range envelope.Data {
		if e.Source == "builtin" {
			seen[e.Name] = true
		}
	}
	if len(seen) != len(builtinAssets) {
		t.Errorf("expected %d builtin assets, saw %d: %v", len(builtinAssets), len(seen), seen)
	}
}

// TestBuiltinSource_SourceListIncludesBuiltin verifies the auto-injected
// builtin source appears in `nd source list` with type "builtin" and alias
// "nd", and that it is never written into the persisted config.
func TestBuiltinSource_SourceListIncludesBuiltin(t *testing.T) {
	env := setupBuiltinEnv(t)

	res := runNDEnv(t, env.env, "--config", env.configPath, "source", "list")
	if res.ExitCode != 0 {
		t.Fatalf("nd source list exit %d, stderr: %s", res.ExitCode, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "builtin") {
		t.Errorf("expected 'builtin' in `nd source list` output, got:\n%s", res.Stdout)
	}

	jres := runNDEnv(t, env.env, "--config", env.configPath, "--json", "source", "list")
	if jres.ExitCode != 0 {
		t.Fatalf("nd --json source list exit %d, stderr: %s", jres.ExitCode, jres.Stderr)
	}
	var envelope struct {
		Status string `json:"status"`
		Data   []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Alias string `json:"alias"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jres.Stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, jres.Stdout)
	}
	var found bool
	for _, s := range envelope.Data {
		if s.ID == "builtin" {
			found = true
			if s.Type != "builtin" {
				t.Errorf("expected builtin source type %q, got %q", "builtin", s.Type)
			}
			if s.Alias != "nd" {
				t.Errorf("expected builtin source alias %q, got %q", "nd", s.Alias)
			}
		}
	}
	if !found {
		t.Errorf("expected a source with id 'builtin' in `nd source list --json`, got:\n%s", jres.Stdout)
	}

	// The builtin entry must never be persisted to config.yaml.
	data, err := os.ReadFile(env.configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), "id: builtin") {
		t.Errorf("builtin source leaked into persisted config:\n%s", data)
	}
}

// TestBuiltinSource_DeploysViaCompiledBinary deploys all six builtin assets
// through the compiled binary into an isolated agent dir and asserts the
// symlinks are created. This proves the embedded assets are deployable
// end-to-end without mutating the developer's real agent config.
func TestBuiltinSource_DeploysViaCompiledBinary(t *testing.T) {
	env := setupBuiltinEnv(t)

	args := []string{"--config", env.configPath, "--json", "--yes", "deploy"}
	for _, a := range builtinAssets {
		args = append(args, a.Ref)
	}
	res := runNDEnv(t, env.env, args...)
	if res.ExitCode != 0 {
		t.Fatalf("nd deploy exit %d, stderr: %s\nstdout: %s", res.ExitCode, res.Stderr, res.Stdout)
	}

	var envelope struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &envelope); err != nil {
		t.Fatalf("invalid deploy JSON: %v\n%s", err, res.Stdout)
	}
	if envelope.Status != "ok" {
		t.Errorf("expected deploy status ok, got %q\n%s", envelope.Status, res.Stdout)
	}

	// Each asset must now be a symlink under the isolated agent global dir.
	for _, a := range builtinAssets {
		link := filepath.Join(env.agentDir, a.LinkRel)
		info, err := os.Lstat(link)
		if err != nil {
			t.Errorf("expected deployed symlink %s: %v", link, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("expected %s to be a symlink, got mode %s", link, info.Mode())
		}
	}

	// Cross-check: `nd list` should now report the six assets as deployed.
	lres := runNDEnv(t, env.env, "--config", env.configPath, "list")
	if lres.ExitCode != 0 {
		t.Fatalf("nd list (post-deploy) exit %d, stderr: %s", lres.ExitCode, lres.Stderr)
	}
	for _, a := range builtinAssets {
		if !strings.Contains(lres.Stdout, a.Name) {
			t.Errorf("expected %q in post-deploy `nd list`, got:\n%s", a.Name, lres.Stdout)
		}
	}
}

// TestBuiltinSource_RemoveIsRefused verifies the removal guard: the compiled
// binary refuses to remove the builtin source and exits non-zero.
func TestBuiltinSource_RemoveIsRefused(t *testing.T) {
	env := setupBuiltinEnv(t)

	// --yes is required to pass the (non-interactive) confirmation and reach
	// the guard in SourceManager.Remove.
	res := runNDEnv(t, env.env, "--config", env.configPath, "source", "remove", "builtin", "--yes")
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit removing builtin, stdout: %s", res.Stdout)
	}
	if !strings.Contains(res.Stderr, "builtin source cannot be removed") {
		t.Errorf("expected removal-guard message in stderr, got: %s", res.Stderr)
	}
}

// TestBuiltinSource_InitYesInitializes exercises `nd init --yes` end-to-end
// through the compiled binary and asserts it initializes cleanly.
//
// The compiled binary resolves the agent global dir for init's deploy step via
// user.Current() (which ignores $HOME), so a real deploy cannot be redirected
// to a temp dir. To avoid mutating a developer's real ~/.claude, this test only
// runs when no real agent dir exists (fresh/CI environment) and the subprocess
// PATH cannot resolve a real agent binary — conditions under which init detects
// no agent and safely skips deployment. The init auto-deploy path itself is
// covered hermetically by cmd/init_cmd_test.go (which injects an isolated
// agent), and deploy of the builtin assets via the binary is covered by
// TestBuiltinSource_DeploysViaCompiledBinary.
func TestBuiltinSource_InitYesInitializes(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		t.Skipf("cannot determine current user home: %v", err)
	}
	realAgentDir := filepath.Join(u.HomeDir, ".claude")
	if _, lerr := os.Lstat(realAgentDir); lerr == nil {
		t.Skipf("real agent dir %s exists; skipping compiled `nd init` to avoid mutating it", realAgentDir)
	}

	env := setupBuiltinEnv(t)
	// Point --config at a path that does not yet exist so `nd init` proceeds.
	freshConfig := filepath.Join(filepath.Dir(env.configPath), "fresh", "config.yaml")

	res := runNDEnv(t, env.env, "--config", freshConfig, "--yes", "--json", "init")
	if res.ExitCode != 0 {
		t.Fatalf("nd init exit %d, stderr: %s\nstdout: %s", res.ExitCode, res.Stderr, res.Stdout)
	}

	var envelope struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &envelope); err != nil {
		t.Fatalf("invalid init JSON: %v\n%s", err, res.Stdout)
	}
	if envelope.Status != "ok" {
		t.Errorf("expected init status ok, got %q", envelope.Status)
	}
	if _, err := os.Stat(freshConfig); err != nil {
		t.Errorf("expected init to create config at %s: %v", freshConfig, err)
	}
	// If an agent happened to be detected, init reports the deploy count.
	if v, ok := envelope.Data["builtin_deployed"]; ok {
		if n, ok := v.(float64); !ok || n < 1 {
			t.Errorf("expected builtin_deployed >= 1 when present, got %v", v)
		}
	}
}
