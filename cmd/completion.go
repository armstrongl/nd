package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCompletionCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for nd.

Available shells: bash, zsh, fish

Run "nd completion <shell> --help" for shell-specific instructions.`,
		Hidden: true,
	}

	cmd.AddCommand(
		newCompletionBashCmd(app),
		newCompletionZshCmd(app),
		newCompletionFishCmd(app),
	)

	return cmd
}

func newCompletionBashCmd(app *App) *cobra.Command {
	var install bool
	var installDir string

	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate bash completion script",
		Long: `Generate bash completion script for nd.

To install completions:
  nd completion bash --install

Or manually:
  nd completion bash > ~/.local/share/bash-completion/completions/nd

Then restart your shell or source the file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			if !install {
				return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
			}
			dir := installDir
			if dir == "" {
				dir = defaultBashCompletionDir()
			}
			return installCompletion(cmd, func(buf *bytes.Buffer) error {
				return rootCmd.GenBashCompletionV2(buf, true)
			}, dir, "nd")
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install to standard location")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "override install directory")
	return cmd
}

func newCompletionZshCmd(app *App) *cobra.Command {
	var install bool
	var installDir string

	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate zsh completion script",
		Long: `Generate zsh completion script for nd.

To install completions:
  nd completion zsh --install

Or manually:
  nd completion zsh > ~/.zfunc/_nd

Then add to ~/.zshrc (if not already present):
  fpath+=~/.zfunc
  autoload -Uz compinit && compinit`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			if !install {
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			}
			dir := installDir
			if dir == "" {
				dir = defaultZshCompletionDir()
			}
			return installCompletion(cmd, func(buf *bytes.Buffer) error {
				return rootCmd.GenZshCompletion(buf)
			}, dir, "_nd")
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install to standard location")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "override install directory")
	return cmd
}

func newCompletionFishCmd(app *App) *cobra.Command {
	var install bool
	var installDir string

	cmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate fish completion script",
		Long: `Generate fish completion script for nd.

To install completions:
  nd completion fish --install

Or manually:
  nd completion fish > ~/.config/fish/completions/nd.fish`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Root()
			if !install {
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			}
			dir := installDir
			if dir == "" {
				dir = defaultFishCompletionDir()
			}
			return installCompletion(cmd, func(buf *bytes.Buffer) error {
				return rootCmd.GenFishCompletion(buf, true)
			}, dir, "nd.fish")
		},
	}
	cmd.Flags().BoolVar(&install, "install", false, "install to standard location")
	cmd.Flags().StringVar(&installDir, "install-dir", "", "override install directory")
	return cmd
}

func defaultBashCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	xdg := filepath.Join(home, ".local", "share", "bash-completion", "completions")
	if info, err := os.Stat(xdg); err == nil && info.IsDir() {
		return xdg
	}
	return filepath.Join(home, ".bash_completion.d")
}

func defaultZshCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".zfunc")
}

func defaultFishCompletionDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "fish", "completions")
}

func installCompletion(cmd *cobra.Command, genFn func(*bytes.Buffer) error, dir, filename string) error {
	if dir == "" {
		return fmt.Errorf("could not determine home directory; use --install-dir to specify a path")
	}

	var buf bytes.Buffer
	if err := genFn(&buf); err != nil {
		return fmt.Errorf("generate completion: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w\n\nYou can install manually instead:\n  nd completion %s > <path>",
			dir, err, cmd.Name())
	}

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w\n\nYou can install manually instead:\n  nd completion %s > <path>",
			path, err, cmd.Name())
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Completion script installed to %s\n", path)
	return nil
}
