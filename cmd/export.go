package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/armstrongl/nd/internal/asset"
	"github.com/armstrongl/nd/internal/export"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/oplog"
)

func newExportCmd(app *App) *cobra.Command {
	var (
		name        string
		description string
		version     string
		author      string
		email       string
		license     string
		output      string
		assets      []string
		source      string
		overwrite   bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export assets as a Claude Code plugin",
		Long: `Export one or more nd-managed assets into the Claude Code plugin format.

Assets are specified with --assets in type/name format (e.g., skills/greeting).
Multiple assets can be comma-separated or the flag repeated.`,
		Example: `  # Export assets as a Claude Code plugin
  nd export --assets skills/greeting,commands/hello --output ./my-plugin

  # Generate a marketplace from plugins
  nd export marketplace --plugins ./plugin-a,./plugin-b --output ./marketplace`,
		Annotations: map[string]string{
			"docs.guides": "getting-started,creating-sources",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			// Determine if we have enough flags for non-interactive mode
			hasName := name != ""
			hasAssets := len(assets) > 0

			// Interactive mode: terminal + missing required flags
			if isTerminal() && (!hasName || !hasAssets) {
				return runExportInteractive(cmd, app, name, description, version, author, email, license, output, source, overwrite)
			}

			// Non-interactive: require --name and --assets
			if !hasName {
				return fmt.Errorf("--name is required (or run interactively in a terminal)")
			}
			if !hasAssets {
				return fmt.Errorf("--assets is required (or run interactively in a terminal)")
			}

			// Parse and validate asset refs
			refs, err := parseAssetRefs(assets)
			if err != nil {
				return withExitCode(nd.ExitInvalidUsage, err)
			}

			// Scan sources
			summary, err := app.ScanIndex()
			if err != nil {
				return fmt.Errorf("scan sources: %w", err)
			}
			index := summary.Index

			// Resolve asset refs to export.AssetRef
			assetRefs, err := resolveExportAssetRefs(index, refs, source)
			if err != nil {
				return err
			}

			// Build output dir
			outputDir := output
			if outputDir == "" {
				outputDir = "./" + name
			}

			cfg := export.ExportConfig{
				Name:        name,
				Version:     version,
				Description: description,
				Author:      export.Author{Name: author, Email: email},
				License:     license,
				OutputDir:   outputDir,
				Assets:      assetRefs,
				Overwrite:   overwrite,
			}

			if app.DryRun {
				if app.JSON {
					return printJSON(w, cfg, true)
				}
				printHuman(w, "[dry-run] would export plugin %q with %d asset(s) to %s\n", name, len(assetRefs), outputDir)
				for _, a := range assetRefs {
					printHuman(w, "[dry-run]   %s/%s\n", a.Type, a.Name)
				}
				return nil
			}

			exporter := &export.PluginExporter{}
			result, err := exporter.Export(cfg)
			if err != nil {
				return err
			}

			// Log operation
			var logAssets []asset.Identity
			for _, a := range assetRefs {
				logAssets = append(logAssets, asset.Identity{
					SourceID: a.SourceID,
					Type:     a.Type,
					Name:     a.Name,
				})
			}
			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpExport,
				Assets:    logAssets,
				Succeeded: len(result.CopiedAssets) + len(result.BundledExtras),
				Failed:    0,
				Detail:    result.PluginDir,
			})

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				printHuman(w, "Exported plugin %q to %s\n", result.PluginName, result.PluginDir)
				printHuman(w, "  %d asset(s) copied\n", len(result.CopiedAssets))
				if len(result.BundledExtras) > 0 {
					printHuman(w, "  %d extra(s) bundled\n", len(result.BundledExtras))
				}
				for _, warn := range result.Warnings {
					printHuman(cmd.ErrOrStderr(), "warning: %s\n", warn)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "plugin name (kebab-case)")
	cmd.Flags().StringVar(&description, "description", "", "plugin description")
	cmd.Flags().StringVar(&version, "version", "1.0.0", "plugin version")
	cmd.Flags().StringVar(&author, "author", "", "author name")
	cmd.Flags().StringVar(&email, "email", "", "author email")
	cmd.Flags().StringVar(&license, "license", "", "SPDX license identifier")
	cmd.Flags().StringVar(&output, "output", "", "output directory (default ./<name>)")
	cmd.Flags().StringSliceVar(&assets, "assets", nil, "assets to export (type/name format, comma-separated)")
	cmd.Flags().StringVar(&source, "source", "", "export only from this source")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing output directory")

	cmd.RegisterFlagCompletionFunc("assets", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		completionInitApp(app)
		summary, err := app.ScanIndex()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		agentAlias := ""
		if ag, err := app.DefaultAgent(); err == nil {
			agentAlias = ag.SourceAlias
		}
		var names []string
		for _, a := range summary.Index.FilterByAgent(agentAlias) {
			if a.Type == nd.AssetPlugin {
				continue
			}
			name := fmt.Sprintf("%s/%s", a.Type, a.Name)
			if toComplete == "" || strings.HasPrefix(name, toComplete) {
				names = append(names, fmt.Sprintf("%s/%s\t%s from %s", a.Type, a.Name, a.Type, a.SourceID))
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("source", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeSourceIDs(app, toComplete)
	})

	// Add marketplace subcommand
	cmd.AddCommand(newExportMarketplaceCmd(app))

	return cmd
}

func newExportMarketplaceCmd(app *App) *cobra.Command {
	var (
		name        string
		description string
		owner       string
		email       string
		plugins     []string
		output      string
		overwrite   bool
	)

	cmd := &cobra.Command{
		Use:   "marketplace",
		Short: "Generate a Claude Code marketplace from exported plugins",
		Long: `Generate a marketplace directory structure from one or more previously exported plugins.

Each --plugins path must point to a directory containing a .claude-plugin/plugin.json file.`,
		Example: `  # Generate marketplace from exported plugins
  nd export marketplace --plugins ./plugin-a,./plugin-b --output ./marketplace`,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if owner == "" {
				return fmt.Errorf("--owner is required")
			}
			if len(plugins) == 0 {
				return fmt.Errorf("--plugins is required (provide paths to exported plugin directories)")
			}

			// Build plugin entries from paths
			entries, err := buildPluginEntries(plugins)
			if err != nil {
				return err
			}

			outputDir := output
			if outputDir == "" {
				outputDir = "./" + name
			}

			cfg := export.MarketplaceConfig{
				Name:        name,
				Description: description,
				Owner:       export.Author{Name: owner, Email: email},
				Plugins:     entries,
				OutputDir:   outputDir,
				Overwrite:   overwrite,
			}

			if app.DryRun {
				if app.JSON {
					return printJSON(w, cfg, true)
				}
				printHuman(w, "[dry-run] would generate marketplace %q with %d plugin(s) to %s\n", name, len(entries), outputDir)
				for _, e := range entries {
					printHuman(w, "[dry-run]   %s (v%s)\n", e.Name, e.Version)
				}
				return nil
			}

			gen := &export.MarketplaceGenerator{}
			result, err := gen.Generate(cfg)
			if err != nil {
				return err
			}

			app.LogOp(oplog.LogEntry{
				Timestamp: time.Now(),
				Operation: oplog.OpExportMarketplace,
				Succeeded: result.PluginCount,
				Failed:    0,
				Detail:    result.MarketplaceDir,
			})

			if app.JSON {
				return printJSON(w, result, false)
			}

			if !app.Quiet {
				printHuman(w, "Generated marketplace %q at %s\n", result.MarketplaceName, result.MarketplaceDir)
				printHuman(w, "  %d plugin(s) included\n", result.PluginCount)
				for _, warn := range result.Warnings {
					printHuman(cmd.ErrOrStderr(), "warning: %s\n", warn)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "marketplace name (kebab-case)")
	cmd.Flags().StringVar(&description, "description", "", "marketplace description")
	cmd.Flags().StringVar(&owner, "owner", "", "marketplace owner name")
	cmd.Flags().StringVar(&email, "email", "", "owner email")
	cmd.Flags().StringSliceVar(&plugins, "plugins", nil, "paths to exported plugin directories")
	cmd.Flags().StringVar(&output, "output", "", "output directory (default ./<name>)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "overwrite existing output directory")

	return cmd
}

// parseAssetRefs validates and splits asset reference strings into (type, name) pairs.
func parseAssetRefs(refs []string) ([][2]string, error) {
	var parsed [][2]string
	for _, ref := range refs {
		if !strings.Contains(ref, "/") {
			return nil, fmt.Errorf("invalid asset reference %q: expected type/name format (e.g., skills/greeting)", ref)
		}
		parts := strings.SplitN(ref, "/", 2)
		assetType := parts[0]
		name := parts[1]

		if name == "" {
			return nil, fmt.Errorf("invalid asset reference %q: asset name cannot be empty", ref)
		}

		if nd.AssetType(assetType) == nd.AssetPlugin {
			return nil, fmt.Errorf("plugins cannot be exported; install via /plugin install instead")
		}

		parsed = append(parsed, [2]string{assetType, name})
	}
	return parsed, nil
}

// resolveExportAssetRefs resolves parsed asset refs against the index and returns export.AssetRef slices.
func resolveExportAssetRefs(index *asset.Index, refs [][2]string, sourceFilter string) ([]export.AssetRef, error) {
	var result []export.AssetRef

	for _, ref := range refs {
		assetType := nd.AssetType(ref[0])
		name := ref[1]

		var found *asset.Asset
		if sourceFilter != "" {
			// Filter by source: look only in assets from that source
			sourceAssets := index.BySource(sourceFilter)
			lowerName := strings.ToLower(name)
			for _, a := range sourceAssets {
				if a.Type == assetType && strings.ToLower(a.Name) == lowerName {
					found = a
					break
				}
			}
			if found == nil {
				return nil, fmt.Errorf("asset %s/%s not found in source %q", assetType, name, sourceFilter)
			}
		} else {
			found = index.SearchByTypeAndName(assetType, name)
			if found == nil {
				return nil, fmt.Errorf("asset %s/%s not found", assetType, name)
			}
		}

		result = append(result, export.AssetRef{
			Type:     found.Type,
			Name:     found.Name,
			SourceID: found.SourceID,
			Path:     found.SourcePath,
			IsDir:    found.IsDir,
		})
	}

	return result, nil
}

// buildPluginEntries reads plugin.json from each plugin directory to build PluginEntry slices.
func buildPluginEntries(pluginPaths []string) ([]export.PluginEntry, error) {
	var entries []export.PluginEntry

	for _, pluginPath := range pluginPaths {
		absPath, err := filepath.Abs(pluginPath)
		if err != nil {
			return nil, fmt.Errorf("resolve plugin path %q: %w", pluginPath, err)
		}

		pluginJSONPath := filepath.Join(absPath, ".claude-plugin", "plugin.json")
		data, err := os.ReadFile(pluginJSONPath)
		if err != nil {
			return nil, fmt.Errorf("read plugin.json from %q: %w", pluginPath, err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("parse plugin.json from %q: %w", pluginPath, err)
		}

		pluginName, _ := parsed["name"].(string)
		if pluginName == "" {
			return nil, fmt.Errorf("plugin.json in %q missing required \"name\" field", pluginPath)
		}

		pluginDesc, _ := parsed["description"].(string)
		pluginVersion, _ := parsed["version"].(string)
		if pluginVersion == "" {
			pluginVersion = "1.0.0"
		}

		var pluginAuthor *export.Author
		if authorMap, ok := parsed["author"].(map[string]any); ok {
			pluginAuthor = &export.Author{
				Name:  stringFromMap(authorMap, "name"),
				Email: stringFromMap(authorMap, "email"),
			}
		}

		entries = append(entries, export.PluginEntry{
			Name:        pluginName,
			Source:      absPath,
			Description: pluginDesc,
			Version:     pluginVersion,
			Author:      pluginAuthor,
		})
	}

	return entries, nil
}

// stringFromMap safely extracts a string from a map[string]any.
func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// runExportInteractive runs the export command in interactive mode using huh forms.
func runExportInteractive(cmd *cobra.Command, app *App, flagName, flagDesc, flagVersion, flagAuthor, flagEmail, flagLicense, flagOutput, flagSource string, flagOverwrite bool) error {
	w := cmd.OutOrStdout()

	// Scan sources first
	summary, err := app.ScanIndex()
	if err != nil {
		return fmt.Errorf("scan sources: %w", err)
	}
	index := summary.Index

	// Build asset choices (exclude plugins)
	agentAlias := ""
	if ag, err := app.DefaultAgent(); err == nil {
		agentAlias = ag.SourceAlias
	}
	allAssets := index.FilterByAgent(agentAlias)
	var choices []huh.Option[string]
	for _, a := range allAssets {
		if a.Type == nd.AssetPlugin {
			continue
		}
		ref := fmt.Sprintf("%s/%s", a.Type, a.Name)
		label := fmt.Sprintf("%s (%s)", ref, a.SourceID)
		choices = append(choices, huh.NewOption(label, ref))
	}

	if len(choices) == 0 {
		return fmt.Errorf("no exportable assets found; add a source with 'nd source add <path>'")
	}

	// Use flag values as defaults where provided
	name := flagName
	description := flagDesc
	version := flagVersion
	author := flagAuthor
	email := flagEmail
	license := flagLicense

	var selectedAssets []string

	// Asset selection form
	assetForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select assets to export").
				Options(choices...).
				Value(&selectedAssets),
		),
	)

	if err := assetForm.Run(); err != nil {
		return fmt.Errorf("asset selection cancelled: %w", err)
	}

	if len(selectedAssets) == 0 {
		return fmt.Errorf("no assets selected")
	}

	// Metadata form
	metaForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Plugin name (kebab-case)").
				Value(&name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					if !export.IsKebabCase(s) {
						return fmt.Errorf("must be kebab-case")
					}
					return nil
				}),
			huh.NewInput().
				Title("Description").
				Value(&description),
			huh.NewInput().
				Title("Version").
				Value(&version),
			huh.NewInput().
				Title("Author name").
				Value(&author),
			huh.NewInput().
				Title("Author email").
				Value(&email),
			huh.NewInput().
				Title("License (SPDX)").
				Value(&license),
		),
	)

	if err := metaForm.Run(); err != nil {
		return fmt.Errorf("metadata entry cancelled: %w", err)
	}

	// Build output dir
	outputDir := flagOutput
	if outputDir == "" {
		outputDir = "./" + name
	}

	// Confirmation
	var confirmed bool
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Export %d asset(s) as plugin %q to %s?", len(selectedAssets), name, outputDir)).
				Value(&confirmed),
		),
	)

	if err := confirmForm.Run(); err != nil {
		return fmt.Errorf("confirmation cancelled: %w", err)
	}

	if !confirmed {
		printHuman(w, "Export cancelled.\n")
		return nil
	}

	// Parse selected asset refs
	refs, err := parseAssetRefs(selectedAssets)
	if err != nil {
		return err
	}

	assetRefs, err := resolveExportAssetRefs(index, refs, flagSource)
	if err != nil {
		return err
	}

	cfg := export.ExportConfig{
		Name:        name,
		Version:     version,
		Description: description,
		Author:      export.Author{Name: author, Email: email},
		License:     license,
		OutputDir:   outputDir,
		Assets:      assetRefs,
		Overwrite:   flagOverwrite,
	}

	if app.DryRun {
		if app.JSON {
			return printJSON(w, cfg, true)
		}
		printHuman(w, "[dry-run] would export plugin %q with %d asset(s) to %s\n", name, len(assetRefs), outputDir)
		for _, a := range assetRefs {
			printHuman(w, "[dry-run]   %s/%s\n", a.Type, a.Name)
		}
		return nil
	}

	exporter := &export.PluginExporter{}
	result, err := exporter.Export(cfg)
	if err != nil {
		return err
	}

	// Log operation
	var logAssets []asset.Identity
	for _, a := range assetRefs {
		logAssets = append(logAssets, asset.Identity{
			SourceID: a.SourceID,
			Type:     a.Type,
			Name:     a.Name,
		})
	}
	app.LogOp(oplog.LogEntry{
		Timestamp: time.Now(),
		Operation: oplog.OpExport,
		Assets:    logAssets,
		Succeeded: len(result.CopiedAssets) + len(result.BundledExtras),
		Failed:    0,
		Detail:    result.PluginDir,
	})

	if app.JSON {
		return printJSON(w, result, false)
	}

	if !app.Quiet {
		printHuman(w, "Exported plugin %q to %s\n", result.PluginName, result.PluginDir)
		printHuman(w, "  %d asset(s) copied\n", len(result.CopiedAssets))
		if len(result.BundledExtras) > 0 {
			printHuman(w, "  %d extra(s) bundled\n", len(result.BundledExtras))
		}
		for _, warn := range result.Warnings {
			printHuman(cmd.ErrOrStderr(), "warning: %s\n", warn)
		}
	}

	return nil
}
