package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/larah/nd/internal/nd"
	"github.com/larah/nd/internal/profile"
	"github.com/spf13/cobra"
)

func newProfileCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage deployment profiles",
		Long:  "Create, list, deploy, and switch between named profiles.",
	}

	cmd.AddCommand(
		newProfileCreateCmd(app),
		newProfileDeleteCmd(app),
		newProfileListCmd(app),
		newProfileDeployCmd(app),
		newProfileSwitchCmd(app),
		newProfileAddAssetCmd(app),
	)

	return cmd
}

func newProfileCreateCmd(app *App) *cobra.Command {
	var (
		assets      string
		fromCurrent bool
		description string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := args[0]

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			now := time.Now().Truncate(time.Second)
			p := profile.Profile{
				Version:     nd.SchemaVersion,
				Name:        name,
				Description: description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}

			if fromCurrent {
				// Build profile from current deployment state
				sstore := app.StateStore()
				st, _, err := sstore.Load()
				if err != nil {
					return fmt.Errorf("load deployment state: %w", err)
				}
				for _, d := range st.Deployments {
					p.Assets = append(p.Assets, profile.ProfileAsset{
						SourceID:  d.SourceID,
						AssetType: d.AssetType,
						AssetName: d.AssetName,
						Scope:     d.Scope,
					})
				}
			} else if assets != "" {
				// Parse --assets flag: "type/name,type/name,..."
				summary, err := app.ScanIndex()
				if err != nil {
					return fmt.Errorf("scan sources: %w", err)
				}
				index := summary.Index

				for _, ref := range strings.Split(assets, ",") {
					ref = strings.TrimSpace(ref)
					if ref == "" {
						continue
					}
					resolved, err := resolveAssetRef(index, ref, "")
					if err != nil {
						return fmt.Errorf("resolve asset %q: %w", ref, err)
					}
					p.Assets = append(p.Assets, profile.ProfileAsset{
						SourceID:  resolved.SourceID,
						AssetType: resolved.Type,
						AssetName: resolved.Name,
						Scope:     app.Scope,
					})
				}
			}

			if err := pstore.CreateProfile(p); err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, p, app.DryRun)
			}
			if !app.Quiet {
				printHuman(w, "Created profile %q with %d assets.\n", name, len(p.Assets))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&assets, "assets", "", "comma-separated list of assets (type/name)")
	cmd.Flags().BoolVar(&fromCurrent, "from-current", false, "create profile from current deployments")
	cmd.Flags().StringVar(&description, "description", "", "profile description")
	return cmd
}

func newProfileDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := args[0]

			profMgr, err := app.ProfileManager()
			if err != nil {
				return err
			}

			if err := profMgr.DeleteProfile(name); err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, map[string]string{"deleted": name}, false)
			}
			if !app.Quiet {
				printHuman(w, "Deleted profile %q.\n", name)
			}
			return nil
		},
	}
}

func newProfileListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			profiles, err := pstore.ListProfiles()
			if err != nil {
				return err
			}

			// Get active profile for indicator
			profMgr, err := app.ProfileManager()
			if err != nil {
				return err
			}
			active, _ := profMgr.ActiveProfile()

			if app.JSON {
				type profileListEntry struct {
					Name        string `json:"name"`
					Description string `json:"description,omitempty"`
					AssetCount  int    `json:"asset_count"`
					Active      bool   `json:"active"`
				}
				entries := make([]profileListEntry, len(profiles))
				for i, p := range profiles {
					entries[i] = profileListEntry{
						Name:        p.Name,
						Description: p.Description,
						AssetCount:  p.AssetCount,
						Active:      p.Name == active,
					}
				}
				return printJSON(w, entries, app.DryRun)
			}

			if len(profiles) == 0 {
				printHuman(w, "No profiles found.\n")
				return nil
			}

			for _, p := range profiles {
				marker := " "
				if p.Name == active {
					marker = "*"
				}
				if p.Description != "" {
					printHuman(w, " %s %-20s %d assets  %s\n", marker, p.Name, p.AssetCount, p.Description)
				} else {
					printHuman(w, " %s %-20s %d assets\n", marker, p.Name, p.AssetCount)
				}
			}
			return nil
		},
	}
}

func newProfileDeployCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "deploy <name>",
		Short: "Deploy all assets in a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			name := args[0]

			profMgr, err := app.ProfileManager()
			if err != nil {
				return err
			}

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			// Verify profile exists
			target, err := pstore.GetProfile(name)
			if err != nil {
				return err
			}

			if app.DryRun {
				if app.JSON {
					return printJSON(w, target, true)
				}
				for _, a := range target.Assets {
					printHuman(w, "[dry-run] would deploy %s/%s from %s\n", a.AssetType, a.AssetName, a.SourceID)
				}
				return nil
			}

			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			result, err := profMgr.DeployProfile(name, eng, summary.Index, app.ProjectRoot)
			if err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				if result.Deployed != nil {
					for _, s := range result.Deployed.Succeeded {
						printHuman(w, "Deployed %s/%s\n", s.Deployment.AssetType, s.Deployment.AssetName)
					}
				}
				for _, m := range result.MissingAssets {
					printHuman(cmd.ErrOrStderr(), "Warning: asset %s/%s not found in sources\n", m.AssetType, m.AssetName)
				}
				printHuman(w, "Profile %q deployed.\n", name)
			}
			return nil
		},
	}
}

func newProfileSwitchCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch from current profile to another",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			targetName := args[0]

			profMgr, err := app.ProfileManager()
			if err != nil {
				return err
			}

			currentName, err := profMgr.ActiveProfile()
			if err != nil {
				return err
			}
			if currentName == "" {
				return fmt.Errorf("no active profile; use 'nd profile deploy <name>' instead")
			}

			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}

			eng, err := app.DeployEngine()
			if err != nil {
				return err
			}

			if app.DryRun {
				pstore, err := app.ProfileStore()
				if err != nil {
					return err
				}
				current, err := pstore.GetProfile(currentName)
				if err != nil {
					return err
				}
				target, err := pstore.GetProfile(targetName)
				if err != nil {
					return err
				}
				diff := profile.ComputeSwitchDiff(current, target)

				if app.JSON {
					return printJSON(w, diff, true)
				}
				for _, a := range diff.Remove {
					printHuman(w, "[dry-run] would remove %s/%s\n", a.AssetType, a.AssetName)
				}
				for _, a := range diff.Deploy {
					printHuman(w, "[dry-run] would deploy %s/%s\n", a.AssetType, a.AssetName)
				}
				printHuman(w, "[dry-run] would switch from %q to %q\n", currentName, targetName)
				return nil
			}

			result, err := profMgr.Switch(currentName, targetName, eng, summary.Index, app.ProjectRoot)
			if err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				if result.Removed != nil {
					for _, s := range result.Removed.Succeeded {
						printHuman(w, "Removed %s/%s\n", s.Identity.Type, s.Identity.Name)
					}
				}
				if result.Deployed != nil {
					for _, s := range result.Deployed.Succeeded {
						printHuman(w, "Deployed %s/%s\n", s.Deployment.AssetType, s.Deployment.AssetName)
					}
				}
				for _, m := range result.MissingAssets {
					printHuman(cmd.ErrOrStderr(), "Warning: asset %s/%s not found in sources\n", m.AssetType, m.AssetName)
				}
				printHuman(w, "Switched from %q to %q.\n", currentName, targetName)
			}
			return nil
		},
	}
}

func newProfileAddAssetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "add-asset <profile> <asset>",
		Short: "Add an asset to an existing profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			profileName := args[0]
			assetRef := args[1]

			pstore, err := app.ProfileStore()
			if err != nil {
				return err
			}

			p, err := pstore.GetProfile(profileName)
			if err != nil {
				return err
			}

			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}

			resolved, err := resolveAssetRef(summary.Index, assetRef, "")
			if err != nil {
				return err
			}

			pa := profile.ProfileAsset{
				SourceID:  resolved.SourceID,
				AssetType: resolved.Type,
				AssetName: resolved.Name,
				Scope:     app.Scope,
			}

			// Check for duplicate asset
			for _, existing := range p.Assets {
				if existing.AssetType == pa.AssetType && existing.AssetName == pa.AssetName {
					return fmt.Errorf("asset %s/%s already exists in profile %q", pa.AssetType, pa.AssetName, profileName)
				}
			}

			p.Assets = append(p.Assets, pa)
			p.UpdatedAt = time.Now().Truncate(time.Second)

			if app.DryRun {
				if app.JSON {
					return printJSON(w, p, true)
				}
				if !app.Quiet {
					printHuman(w, "[dry-run] would add %s/%s to profile %q.\n", resolved.Type, resolved.Name, profileName)
				}
				return nil
			}

			if err := pstore.UpdateProfile(*p); err != nil {
				return err
			}

			if app.JSON {
				return printJSON(w, p, false)
			}
			if !app.Quiet {
				printHuman(w, "Added %s/%s to profile %q.\n", resolved.Type, resolved.Name, profileName)
			}
			return nil
		},
	}
}
