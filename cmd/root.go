package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/tui"
	"github.com/armstrongl/nd/internal/updater"
	"github.com/armstrongl/nd/internal/version"
)

// NewRootCmd creates the root command with all global flags and subcommands.
func NewRootCmd(app *App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "nd",
		Version:       version.String(),
		Short:         "Napoleon Dynamite - coding agent asset manager",
		Long:          "nd manages coding agent assets (skills, commands, rules, etc.) via symlink deployment.",
		Example: `  # Deploy an asset
  nd deploy skills/greeting

  # List available assets
  nd list --type skills

  # Check deployment health
  nd doctor

  # Open the TUI
  nd`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return persistentPreRun(cmd, app)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return persistentPostRun(cmd, app)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Launch TUI when running interactively with no flags that conflict.
			if isTerminal() && !app.Verbose && !app.Quiet && !app.JSON {
				return tui.Run(app)
			}
			return cmd.Help()
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

	rootCmd.RegisterFlagCompletionFunc("scope", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"global\tDeploy to ~/.claude/", "project\tDeploy to .claude/ in project"}, cobra.ShellCompDirectiveNoFileComp
	})

	// Disable Cobra's default completion command; we provide our own with --install support.
	rootCmd.CompletionOptions.DisableDefaultCmd = true

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
		newCompletionCmd(app),
		newExportCmd(app),
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

// persistentPostRun runs after every successful command. It shows an update
// notice when a newer version is available and the user installed nd via
// Homebrew, then kicks off a background cache refresh.
func persistentPostRun(cmd *cobra.Command, app *App) error {
	if app.Quiet || app.JSON {
		return nil
	}
	if version.Version == "dev" || !updater.IsBrewInstall() {
		return nil
	}
	cacheDir := filepath.Dir(app.ConfigPath)
	if cacheDir == "" || cacheDir == "." {
		return nil
	}
	latest, _ := updater.CheckCached(cacheDir)
	if latest != "" && updater.IsNewer(latest, version.Version) {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"\nA new version of nd is available: v%s (you have %s)\nTo update: brew upgrade nd\n\n",
			latest, version.Version,
		)
	}
	updater.RefreshAsync(cacheDir)
	return nil
}


func persistentPreRun(cmd *cobra.Command, app *App) error {
	// Expand ~ in config path
	if strings.HasPrefix(app.ConfigPath, "~/") {
		if u, err := user.Current(); err == nil {
			app.ConfigPath = filepath.Join(u.HomeDir, app.ConfigPath[2:])
		}
	}

	// Derive backup dir from config dir
	app.BackupDir = filepath.Join(filepath.Dir(app.ConfigPath), "backups")

	// Offer init when config doesn't exist and command needs it
	if _, err := os.Stat(app.ConfigPath); err != nil {
		if os.IsNotExist(err) {
			if needsInit(cmd) {
				if err := offerInit(cmd, app); err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("stat config path %q: %w", app.ConfigPath, err)
		}
	}

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

// needsInit returns true if the command requires an initialized config.
// Commands that work without config are exempt. Walks the command ancestry
// so subcommands (e.g. "completion bash") and Cobra internals are also exempt.
func needsInit(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "init", "version", "completion", "help",
			"__complete", "__completeNoDesc":
			return false
		case "nd":
			// The root "nd" command itself doesn't need init,
			// but subcommands under it do.
			if c == cmd {
				return false
			}
		}
	}
	return true
}

// offerInit warns the user that nd is not initialized and offers to run init.
// In interactive mode, prompts for confirmation. In non-interactive mode or
// if declined, prints a hint and continues. Skipped during --dry-run.
func offerInit(cmd *cobra.Command, app *App) error {
	if app.DryRun {
		printHuman(cmd.ErrOrStderr(), "nd is not initialized (dry-run: skipping auto-init).\n")
		printHuman(cmd.ErrOrStderr(), "Run 'nd init' to get started.\n")
		return nil
	}

	w := cmd.ErrOrStderr()
	printHuman(w, "nd is not initialized.\n")

	shouldInit := app.Yes
	if !shouldInit && isTerminal() {
		confirmed, err := confirm(cmd.InOrStdin(), w, "Run nd init now?", false)
		if err == nil && confirmed {
			shouldInit = true
		}
	}

	if shouldInit {
		configDir, err := runInitSetup(cmd, app)
		if err != nil {
			return err
		}
		_, err = deployBuiltinAssets(cmd, app, configDir, initRegistry(app), app.initAgent)
		if err != nil {
			return err
		}
		printHuman(w, "\n")
		return nil
	}

	printHuman(w, "Run 'nd init' to get started.\n")
	return nil
}

// defaultConfigPath returns the default config file path.
func defaultConfigPath() string {
	if u, err := user.Current(); err == nil {
		return filepath.Join(u.HomeDir, ".config", "nd", "config.yaml")
	}
	return "~/.config/nd/config.yaml"
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
