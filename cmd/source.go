package cmd

import (
	"fmt"
	"strings"

	"github.com/larah/nd/internal/asset"
	"github.com/larah/nd/internal/deploy"
	"github.com/larah/nd/internal/sourcemanager"
	"github.com/spf13/cobra"
)

func newSourceCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Manage asset sources",
		Long:  "Add, remove, and list asset source directories.",
	}

	cmd.AddCommand(
		newSourceAddCmd(app),
		newSourceRemoveCmd(app),
		newSourceListCmd(app),
	)

	return cmd
}

func newSourceAddCmd(app *App) *cobra.Command {
	var alias string

	cmd := &cobra.Command{
		Use:   "add <path|url>",
		Short: "Register a new asset source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sm, err := app.SourceManager()
			if err != nil {
				return err
			}

			target := args[0]
			w := cmd.OutOrStdout()

			// Detect git vs local
			isGit := strings.Contains(target, "://") ||
				strings.HasPrefix(target, "git@") ||
				isGitHubShorthand(target)

			if isGit {
				src, err := sm.AddGit(target, alias)
				if err != nil {
					return err
				}
				if app.JSON {
					return printJSON(w, src, app.DryRun)
				}
				if !app.Quiet {
					displayName := src.ID
					if src.Alias != "" {
						displayName = fmt.Sprintf("%s (%s)", src.Alias, src.ID)
					}
					printHuman(w, "Added git source %s\n  URL:  %s\n  Path: %s\n", displayName, src.URL, src.Path)
				}
				return nil
			}

			src, err := sm.AddLocal(target, alias)
			if err != nil {
				return err
			}
			if app.JSON {
				return printJSON(w, src, app.DryRun)
			}
			if !app.Quiet {
				displayName := src.ID
				if src.Alias != "" {
					displayName = fmt.Sprintf("%s (%s)", src.Alias, src.ID)
				}
				printHuman(w, "Added local source %s\n  Path: %s\n", displayName, src.Path)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&alias, "alias", "", "human-readable alias for the source")
	return cmd
}

func newSourceRemoveCmd(app *App) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove <source-id>",
		Short: "Remove a registered source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceID := args[0]
			sm, err := app.SourceManager()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()

			// Verify source exists
			var found bool
			for _, s := range sm.Sources() {
				if s.ID == sourceID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("source %q not found", sourceID)
			}

			// Check for deployed assets from this source
			var deployedCount int
			eng, engErr := app.DeployEngine()
			if engErr == nil {
				entries, err := eng.Status()
				if err == nil {
					for _, e := range entries {
						if e.Deployment.SourceID == sourceID {
							deployedCount++
						}
					}
				}
			}

			if deployedCount > 0 && !force {
				if app.JSON {
					return fmt.Errorf("source %q has %d deployed assets; use --force to remove", sourceID, deployedCount)
				}
				choices := []string{
					"Remove source and all deployed assets",
					"Remove source only (orphan deployed assets)",
					"Cancel",
				}
				choice, err := promptChoice(
					cmd.InOrStdin(), w,
					fmt.Sprintf("Source %q has %d deployed assets. What would you like to do?", sourceID, deployedCount),
					choices,
				)
				if err != nil {
					return err
				}

				switch choice {
				case choices[0]:
					if err := removeSourceDeployments(eng, sourceID); err != nil {
						return fmt.Errorf("remove deployed assets: %w", err)
					}
				case choices[1]:
					// Orphan — deployments stay
				case choices[2]:
					if !app.Quiet {
						printHuman(w, "Cancelled.\n")
					}
					return nil
				}
			} else if deployedCount > 0 && force {
				if err := removeSourceDeployments(eng, sourceID); err != nil {
					return fmt.Errorf("remove deployed assets: %w", err)
				}
			}

			if err := sm.Remove(sourceID); err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, map[string]string{
					"removed": sourceID,
				}, app.DryRun)
			}
			if !app.Quiet {
				printHuman(w, "Removed source %q\n", sourceID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation and remove deployed assets")
	return cmd
}

func newSourceListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sm, err := app.SourceManager()
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			sources := sm.Sources()

			if app.JSON {
				type sourceInfo struct {
					ID         string `json:"id"`
					Type       string `json:"type"`
					Path       string `json:"path"`
					URL        string `json:"url,omitempty"`
					Alias      string `json:"alias,omitempty"`
					AssetCount int    `json:"asset_count"`
				}
				list := make([]sourceInfo, len(sources))
				for i, s := range sources {
					list[i] = sourceInfo{
						ID:         s.ID,
						Type:       string(s.Type),
						Path:       s.Path,
						URL:        s.URL,
						Alias:      s.Alias,
						AssetCount: countSourceAssets(s.ID, s.Path),
					}
				}
				return printJSON(w, list, app.DryRun)
			}

			if len(sources) == 0 {
				printHuman(w, "No sources registered. Use 'nd source add <path>' to add one.\n")
				return nil
			}

			for _, s := range sources {
				count := countSourceAssets(s.ID, s.Path)
				displayName := s.ID
				if s.Alias != "" {
					displayName = fmt.Sprintf("%s (%s)", s.Alias, s.ID)
				}
				printHuman(w, "%-30s  %-6s  %d assets  %s\n",
					displayName, s.Type, count, s.Path)
			}
			return nil
		},
	}
}

// isGitHubShorthand detects "owner/repo" patterns.
func isGitHubShorthand(s string) bool {
	parts := strings.SplitN(s, "/", 3)
	if len(parts) != 2 {
		return false
	}
	return !strings.Contains(parts[0], ".") && !strings.HasPrefix(s, "/")
}

// countSourceAssets scans a source and returns the total asset count.
func countSourceAssets(sourceID, path string) int {
	result := sourcemanager.ScanSource(sourceID, path)
	return len(result.Assets)
}

// removeSourceDeployments removes all deployed assets from a given source.
func removeSourceDeployments(eng *deploy.Engine, sourceID string) error {
	entries, err := eng.Status()
	if err != nil {
		return err
	}

	var reqs []deploy.RemoveRequest
	for _, e := range entries {
		if e.Deployment.SourceID == sourceID {
			reqs = append(reqs, deploy.RemoveRequest{
				Identity: asset.Identity{
					SourceID: e.Deployment.SourceID,
					Type:     e.Deployment.AssetType,
					Name:     e.Deployment.AssetName,
				},
				Scope:       e.Deployment.Scope,
				ProjectRoot: e.Deployment.ProjectPath,
			})
		}
	}
	if len(reqs) == 0 {
		return nil
	}

	result, err := eng.RemoveBulk(reqs)
	if err != nil {
		return err
	}
	if len(result.Failed) > 0 {
		return fmt.Errorf("failed to remove %d assets", len(result.Failed))
	}
	return nil
}
