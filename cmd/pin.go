package cmd

import (
	"fmt"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/spf13/cobra"
)

func newPinCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin <asset>",
		Short: "Pin an asset to prevent profile switches from removing it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setAssetOrigin(cmd, app, args[0], nd.OriginPinned, "Pinned")
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeDeployedAssets(app, toComplete)
	}
	return cmd
}

func newUnpinCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpin <asset>",
		Short: "Unpin an asset, allowing profile switches to manage it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return setAssetOrigin(cmd, app, args[0], nd.OriginManual, "Unpinned")
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeDeployedAssets(app, toComplete)
	}
	return cmd
}

func setAssetOrigin(cmd *cobra.Command, app *App, ref string, origin nd.DeployOrigin, verb string) error {
	w := cmd.OutOrStdout()

	eng, err := app.DeployEngine()
	if err != nil {
		return err
	}

	entries, err := eng.Status()
	if err != nil {
		return fmt.Errorf("load status: %w", err)
	}

	dep, err := findDeployedAsset(entries, ref, app.Scope, app.ProjectRoot)
	if err != nil {
		return withExitCode(nd.ExitInvalidUsage, err)
	}

	if err := eng.SetOrigin(dep.Deployment.Identity(), app.Scope, app.ProjectRoot, origin); err != nil {
		return err
	}

	if app.JSON {
		return printJSON(w, map[string]string{
			"asset":  fmt.Sprintf("%s/%s", dep.Deployment.AssetType, dep.Deployment.AssetName),
			"origin": string(origin),
		}, app.DryRun)
	}
	if !app.Quiet {
		printHuman(w, "%s %s/%s\n", verb, dep.Deployment.AssetType, dep.Deployment.AssetName)
	}
	return nil
}
