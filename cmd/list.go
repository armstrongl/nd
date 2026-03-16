package cmd

import (
	"fmt"
	"strings"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/spf13/cobra"
)

func newListCmd(app *App) *cobra.Command {
	var (
		assetType string
		sourceID  string
		pattern   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available assets from all sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}
			index := summary.Index

			// Print conflict warnings
			for _, c := range index.Conflicts() {
				if !app.Quiet {
					printHuman(cmd.ErrOrStderr(), "warning: %s/%s exists in both %s and %s (using %s)\n",
						c.Type, c.Name, c.Winner, c.Loser, c.Winner)
				}
			}

			// Get deployment status for cross-referencing
			var deployedSet map[string]string // "type/name" -> "scope"
			eng, engErr := app.DeployEngine()
			if engErr == nil {
				entries, err := eng.Status()
				if err == nil {
					deployedSet = make(map[string]string)
					for _, e := range entries {
						key := fmt.Sprintf("%s/%s", e.Deployment.AssetType, e.Deployment.AssetName)
						deployedSet[key] = string(e.Deployment.Scope)
					}
				}
			}

			all := index.All()

			// Apply filters
			type listEntry struct {
				Type     string `json:"type"`
				Name     string `json:"name"`
				Source   string `json:"source"`
				Status   string `json:"status"`
				IsDir    bool   `json:"is_dir"`
			}

			var entries []listEntry
			for _, a := range all {
				if assetType != "" && string(a.Type) != assetType {
					continue
				}
				if sourceID != "" && a.SourceID != sourceID {
					continue
				}
				if pattern != "" && !strings.Contains(strings.ToLower(a.Name), strings.ToLower(pattern)) {
					continue
				}

				status := "available"
				key := fmt.Sprintf("%s/%s", a.Type, a.Name)
				if deployedSet != nil {
					if _, ok := deployedSet[key]; ok {
						status = "deployed"
					}
				}

				entries = append(entries, listEntry{
					Type:   string(a.Type),
					Name:   a.Name,
					Source: a.SourceID,
					Status: status,
					IsDir:  a.IsDir,
				})
			}

			if app.JSON {
				return printJSON(w, entries, app.DryRun)
			}

			if len(entries) == 0 {
				printHuman(w, "No assets found.\n")
				return nil
			}

			for _, e := range entries {
				marker := " "
				if e.Status == "deployed" {
					marker = "*"
				}
				printHuman(w, "%s %-15s  %-30s  %-15s\n", marker, e.Type, e.Name, e.Source)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&assetType, "type", "", "filter by asset type")
	cmd.Flags().StringVar(&sourceID, "source", "", "filter by source ID")
	cmd.Flags().StringVar(&pattern, "pattern", "", "filter by name pattern")
	return cmd
}

// parseAssetType validates an asset type string. Used by commands with --type flag.
func parseAssetType(s string) (nd.AssetType, error) {
	t := nd.AssetType(s)
	for _, valid := range nd.AllAssetTypes() {
		if t == valid {
			return t, nil
		}
	}
	return "", fmt.Errorf("invalid asset type %q", s)
}
