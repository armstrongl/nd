package cmd

import (
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
			_ = assets
			_ = fromCurrent
			_ = description
			return nil // TODO: implement
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
			return nil // TODO: implement
		},
	}
}

func newProfileListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}

func newProfileDeployCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "deploy <name>",
		Short: "Deploy all assets in a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}

func newProfileSwitchCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch from current profile to another",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}

func newProfileAddAssetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "add-asset <profile> <asset>",
		Short: "Add an asset to an existing profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil // TODO: implement
		},
	}
}
