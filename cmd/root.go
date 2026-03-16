package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui"
	tuiapp "github.com/armstrongl/nd/internal/tui/app"
)

// NewRootCmd creates the root command with all global flags and subcommands.
func NewRootCmd(app *App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "nd",
		Short:         "Napoleon Dynamite — coding agent asset manager",
		Long:          "nd manages coding agent assets (skills, commands, rules, etc.) via symlink deployment.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return persistentPreRun(cmd, app)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(app)
		},
	}

	// Global persistent flags
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&app.ConfigPath, "config", "", defaultConfigPath(), "path to config file")
	pf.StringVarP((*string)(&app.Scope), "scope", "s", string(nd.ScopeGlobal), "deployment scope (global|project)")
	pf.BoolVar(&app.DryRun, "dry-run", false, "show what would happen without making changes")
	pf.BoolVarP(&app.Verbose, "verbose", "v", false, "verbose output to stderr")
	pf.BoolVarP(&app.Quiet, "quiet", "q", false, "suppress non-error output")
	pf.BoolVar(&app.JSON, "json", false, "output in JSON format")
	pf.BoolVar(&app.NoColor, "no-color", false, "disable colored output")
	pf.BoolVarP(&app.Yes, "yes", "y", false, "skip confirmation prompts")

	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")

	// Register subcommands
	rootCmd.AddCommand(
		newVersionCmd(app),
		newSourceCmd(app),
		newDeployCmd(app),
		newRemoveCmd(app),
		newListCmd(app),
		newStatusCmd(app),
		newPinCmd(app),
		newUnpinCmd(app),
		newSyncCmd(app),
		newDoctorCmd(app),
		newProfileCmd(app),
		newSnapshotCmd(app),
		newInitCmd(app),
		newSettingsCmd(app),
		newUninstallCmd(app),
	)

	return rootCmd
}

// Execute runs the root command and returns an exit code.
func Execute() int {
	app := &App{}
	rootCmd := NewRootCmd(app)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if exitCode, ok := exitCodeFromError(err); ok {
			return exitCode
		}
		return nd.ExitError
	}
	return nd.ExitSuccess
}

// persistentPreRun resolves config path and scope before any command runs.
func persistentPreRun(cmd *cobra.Command, app *App) error {
	// Expand ~ in config path
	if strings.HasPrefix(app.ConfigPath, "~/") {
		if u, err := user.Current(); err == nil {
			app.ConfigPath = filepath.Join(u.HomeDir, app.ConfigPath[2:])
		}
	}

	// Derive backup dir from config dir
	app.BackupDir = filepath.Join(filepath.Dir(app.ConfigPath), "backups")

	// Validate scope
	switch app.Scope {
	case nd.ScopeGlobal, nd.ScopeProject:
		// valid
	default:
		return fmt.Errorf("invalid scope %q: must be 'global' or 'project'", app.Scope)
	}

	// Resolve project root when scope is project
	if app.Scope == nd.ScopeProject {
		if _, err := app.ResolveProjectRoot(); err != nil {
			return err
		}
	}

	// Check NO_COLOR env var
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		app.NoColor = true
	}

	return nil
}

// defaultConfigPath returns the default config file path.
func defaultConfigPath() string {
	if u, err := user.Current(); err == nil {
		return filepath.Join(u.HomeDir, ".config", "nd", "config.yaml")
	}
	return "~/.config/nd/config.yaml"
}

// runTUI launches the interactive TUI dashboard.
func runTUI(app *App) error {
	eng, err := app.DeployEngine()
	if err != nil {
		return fmt.Errorf("init deploy engine: %w", err)
	}
	prof, err := app.ProfileManager()
	if err != nil {
		return fmt.Errorf("init profile manager: %w", err)
	}
	src, err := app.SourceManager()
	if err != nil {
		return fmt.Errorf("init source manager: %w", err)
	}
	reg, err := app.AgentRegistry()
	if err != nil {
		return fmt.Errorf("init agent registry: %w", err)
	}

	// Build profile adapter with pre-bound deps
	indexFn := func() *asset.Index {
		summary, err := app.ScanIndex()
		if err != nil || summary == nil {
			return nil
		}
		return summary.Index
	}
	projectRoot, _ := app.ResolveProjectRoot()
	adapter := tui.NewProfileAdapter(prof, eng, indexFn, projectRoot)

	hasProjectDir := app.Scope == nd.ScopeProject
	resolver := func() (string, error) { return app.ResolveProjectRoot() }

	return tuiapp.Run(eng, adapter, src, reg, hasProjectDir, resolver)
}

// exitError wraps an error with an exit code.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// withExitCode wraps an error with a specific exit code.
func withExitCode(code int, err error) error {
	return &exitError{code: code, err: err}
}

// exitCodeFromError extracts the exit code from an exitError.
func exitCodeFromError(err error) (int, bool) {
	var ee *exitError
	if ok := errorAs(err, &ee); ok {
		return ee.code, true
	}
	return 0, false
}

// errorAs is a wrapper for errors.As to avoid import cycle issues in tests.
func errorAs(err error, target interface{}) bool {
	// Use type assertion chain since we control the types
	type unwrapper interface{ Unwrap() error }
	for err != nil {
		if ee, ok := err.(*exitError); ok {
			if t, ok := target.(**exitError); ok {
				*t = ee
				return true
			}
		}
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
