package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/output"
	"github.com/spf13/cobra"
)

// printJSON marshals a JSONResponse envelope to the writer.
func printJSON(w io.Writer, data interface{}, dryRun bool) error {
	resp := output.JSONResponse{
		Status: "ok",
		Data:   data,
	}
	if dryRun {
		resp.DryRun = true
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// printJSONError marshals a JSON error response to the writer.
func printJSONError(w io.Writer, errs []output.JSONError) error {
	status := "error"
	if len(errs) > 1 {
		status = "partial"
	}
	resp := output.JSONResponse{
		Status: status,
		Errors: errs,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// printHuman writes formatted output to the writer.
func printHuman(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...)
}

// delegateToSubcommand runs the parent command's named subcommand (for example
// "list" or "edit") when the parent is invoked with no subcommand of its own,
// so bare "nd source" behaves like "nd source list". Cobra propagates the
// parent's IO streams to the child, so the delegated command writes to the same
// output. A leftover positional argument means the user typed an unrecognized
// subcommand, which is surfaced as an unknown-command error rather than being
// silently swallowed by the delegation.
func delegateToSubcommand(cmd *cobra.Command, args []string, name string) error {
	if len(args) > 0 {
		return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
	}
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c.RunE(c, nil)
		}
	}
	return fmt.Errorf("no %q subcommand available for %q", name, cmd.CommandPath())
}

// confirm prompts for yes/no confirmation.
// Returns true immediately if yesFlag is set.
// Returns an error if stdin is not a terminal (piped input).
func confirm(r io.Reader, w io.Writer, prompt string, yesFlag bool) (bool, error) {
	if yesFlag {
		return true, nil
	}
	if !isTerminal() {
		return false, fmt.Errorf("confirmation required but stdin is not a terminal (use --yes to skip)")
	}

	fmt.Fprintf(w, "%s [y/N] ", prompt)
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false, nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

// promptChoice presents numbered choices and returns the selected option.
// Returns an error if stdin is not a terminal.
func promptChoice(r io.Reader, w io.Writer, prompt string, choices []string) (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("interactive choice required but stdin is not a terminal")
	}

	fmt.Fprintln(w, prompt)
	for i, c := range choices {
		fmt.Fprintf(w, "  %d) %s\n", i+1, c)
	}
	fmt.Fprintf(w, "Choice [1-%d]: ", len(choices))

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	n, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || n < 1 || n > len(choices) {
		return "", fmt.Errorf("invalid choice: %s", scanner.Text())
	}
	return choices[n-1], nil
}

// promptDeployBuiltin presents the built-in asset deploy prompt used by
// `nd init`. It offers three actions and defaults to Yes:
//   - empty input (Enter), "y", or "yes" -> deploy all
//   - "n" or "no"                         -> skip
//   - "l" or "list"                       -> print each asset as "<type>/<name>",
//     then re-prompt
//
// Unrecognized input re-prompts. Like confirm, it returns an error when stdin
// is not a terminal so callers can fall back to the non-interactive skip path.
func promptDeployBuiltin(r io.Reader, w io.Writer, prompt string, assets []asset.Asset) (bool, error) {
	if !isTerminal() {
		return false, fmt.Errorf("confirmation required but stdin is not a terminal (use --yes to skip)")
	}

	scanner := bufio.NewScanner(r)
	for {
		fmt.Fprintf(w, "%s [Y/n/list] ", prompt)
		if !scanner.Scan() {
			return false, nil
		}
		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		case "l", "list":
			for _, a := range assets {
				fmt.Fprintf(w, "%s/%s\n", a.Type, a.Name)
			}
		}
	}
}

// extractChoiceNames strips tab-separated descriptions from completion strings.
// Input: ["skills/greeting\tglobal from src", "commands/hello.md\tglobal from src"]
// Output: ["skills/greeting", "commands/hello.md"]
func extractChoiceNames(completions []string) []string {
	names := make([]string, len(completions))
	for i, c := range completions {
		if idx := strings.IndexByte(c, '\t'); idx >= 0 {
			names[i] = c[:idx]
		} else {
			names[i] = c
		}
	}
	return names
}

// isTerminal checks if stdin is a terminal. It is a package variable (not a
// plain function) so tests can override it to exercise interactive,
// prompt-gated code paths.
var isTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// latestAutoSnapshot returns the name of the most recent auto-snapshot, or "".
func latestAutoSnapshot(app *App) string {
	pstore, err := app.ProfileStore()
	if err != nil {
		return ""
	}
	snapshots, err := pstore.ListSnapshots()
	if err != nil {
		return ""
	}
	var latest string
	var latestTime time.Time
	for _, s := range snapshots {
		if s.Auto && s.CreatedAt.After(latestTime) {
			latest = s.Name
			latestTime = s.CreatedAt
		}
	}
	return latest
}

// completionInitApp does lightweight App initialization for completion contexts.
// PersistentPreRunE is not guaranteed to run during shell completion, so this
// handles the essential setup: expanding ~ in ConfigPath and deriving BackupDir.
// This function is idempotent and safe to call multiple times.
func completionInitApp(app *App) {
	if strings.HasPrefix(app.ConfigPath, "~/") {
		if u, err := user.Current(); err == nil {
			app.ConfigPath = filepath.Join(u.HomeDir, app.ConfigPath[2:])
		}
	}
	if app.BackupDir == "" {
		app.BackupDir = filepath.Join(filepath.Dir(app.ConfigPath), "backups")
	}
}

// completeDeployedAssets returns names of deployed assets for shell completion.
// NOTE: Returns ALL deployed assets regardless of scope because PersistentPreRunE
// hasn't run during completion, so scope defaults to "global". This is acceptable
// since completions are advisory.
func completeDeployedAssets(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	eng, err := app.DeployEngine()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	entries, err := eng.Status()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, e := range entries {
		d := e.Deployment
		name := fmt.Sprintf("%s/%s", d.AssetType, d.AssetName)
		if toComplete == "" || strings.HasPrefix(name, toComplete) || strings.HasPrefix(string(d.AssetName), toComplete) {
			names = append(names, fmt.Sprintf("%s/%s\t%s from %s", d.AssetType, d.AssetName, d.Scope, d.SourceID))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeProfileNames returns profile names for shell completion.
func completeProfileNames(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	pstore, err := app.ProfileStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	profiles, err := pstore.ListProfiles()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, p := range profiles {
		if toComplete == "" || strings.HasPrefix(p.Name, toComplete) {
			desc := fmt.Sprintf("%d assets", p.AssetCount)
			if p.Description != "" {
				desc = p.Description
			}
			names = append(names, fmt.Sprintf("%s\t%s", p.Name, desc))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeSnapshotNames returns snapshot names for shell completion.
func completeSnapshotNames(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	pstore, err := app.ProfileStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	snapshots, err := pstore.ListSnapshots()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, s := range snapshots {
		if toComplete == "" || strings.HasPrefix(s.Name, toComplete) {
			desc := fmt.Sprintf("%d deployments", s.DeploymentCount)
			if s.Auto {
				desc += " (auto)"
			}
			names = append(names, fmt.Sprintf("%s\t%s", s.Name, desc))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeSourceIDs returns source IDs for shell completion.
func completeSourceIDs(app *App, toComplete string) ([]string, cobra.ShellCompDirective) {
	completionInitApp(app)
	sm, err := app.SourceManager()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, s := range sm.Sources() {
		if toComplete == "" || strings.HasPrefix(s.ID, toComplete) {
			desc := string(s.Type)
			if s.Alias != "" {
				desc = s.Alias
			}
			names = append(names, fmt.Sprintf("%s\t%s", s.ID, desc))
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
