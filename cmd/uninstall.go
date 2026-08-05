package cmd

import (
	"fmt"
	"time"

	"github.com/armstrongl/nd/internal/deploy"
	"github.com/armstrongl/nd/internal/oplog"
	"github.com/spf13/cobra"
)

func newUninstallCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove all nd-managed symlinks and optionally config",
		Example: `  # Remove all nd-managed symlinks
  nd uninstall

  # Skip confirmation prompt
  nd uninstall --yes`,
		Annotations: map[string]string{
			"docs.guides": "getting-started,troubleshooting",
		},
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			sstore := app.StateStore()
			st, _, err := sstore.Load()
			if err != nil {
				return fmt.Errorf("load deployment state: %w", err)
			}

			plan := deploy.UninstallPlan{
				Symlinks:     st.Deployments,
				SymlinkCount: len(st.Deployments),
			}

			if len(st.Deployments) == 0 {
				if app.JSON {
					return printJSON(w, plan, app.DryRun)
				}
				printHuman(w, "No deployments to remove.\n")
				return nil
			}

			if app.DryRun {
				if app.JSON {
					return printJSON(w, plan, true)
				}
				printHuman(w, "[dry-run] would remove %d symlinks:\n", len(st.Deployments))
				for _, d := range st.Deployments {
					printHuman(w, "  %s/%s (%s)\n", d.AssetType, d.AssetName, d.LinkPath)
				}
				return nil
			}

			// Confirm before proceeding
			ok, err := confirm(cmd.InOrStdin(), w,
				fmt.Sprintf("Remove all %d nd-managed symlinks?", len(st.Deployments)),
				app.Yes)
			if err != nil {
				return err
			}
			if !ok {
				if !app.Quiet && !app.JSON {
					printHuman(w, "Aborted.\n")
				}
				return nil
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			removeReqs := make([]deploy.RemoveRequest, len(st.Deployments))
			for i, d := range st.Deployments {
				// Target the exact recorded deployment's agent rather than relying
				// on the any-agent fallback. Empty d.Agent maps to "claude-code"
				// per the v1→v2 migration rule in removeOne.
				agentName := d.Agent
				if agentName == "" {
					agentName = "claude-code"
				}
				removeReqs[i] = deploy.RemoveRequest{
					Identity:    d.Identity(),
					Scope:       d.Scope,
					ProjectRoot: d.ProjectPath,
					Agent:       agentName,
				}
			}

			result, err := eng.RemoveBulk(removeReqs)
			if err != nil {
				return fmt.Errorf("remove deployments: %w", err)
			}

			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpUninstall,
				Succeeded: len(result.Succeeded),
				Failed:    len(result.Failed),
			})

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				for _, s := range result.Succeeded {
					printHuman(w, "Removed %s/%s\n", s.Identity.Type, s.Identity.Name)
				}
				for _, f := range result.Failed {
					printHuman(cmd.ErrOrStderr(), "Failed: %s/%s: %v\n", f.Identity.Type, f.Identity.Name, f.Err)
				}
				printHuman(w, "Uninstall complete: %d removed.\n", len(result.Succeeded))
			}
			return nil
		},
	}
}
