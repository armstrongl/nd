// internal/export/types_test.go
package export_test

import (
	"strings"
	"testing"

	"github.com/armstrongl/nd/internal/export"
	"github.com/armstrongl/nd/internal/nd"
)

func TestExportConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  export.ExportConfig
		wantErr string
	}{
		{
			name:    "valid minimal",
			config:  export.ExportConfig{Name: "my-plugin", OutputDir: "/tmp/out", Assets: []export.AssetRef{{Type: nd.AssetSkill, Name: "foo", Path: "/src/foo"}}},
			wantErr: "",
		},
		{
			name:    "empty name",
			config:  export.ExportConfig{OutputDir: "/tmp/out", Assets: []export.AssetRef{{Type: nd.AssetSkill, Name: "foo", Path: "/src/foo"}}},
			wantErr: "name is required",
		},
		{
			name:    "not kebab-case",
			config:  export.ExportConfig{Name: "My Plugin", OutputDir: "/tmp/out", Assets: []export.AssetRef{{Type: nd.AssetSkill, Name: "foo", Path: "/src/foo"}}},
			wantErr: "kebab-case",
		},
		{
			name:    "no assets",
			config:  export.ExportConfig{Name: "my-plugin", OutputDir: "/tmp/out"},
			wantErr: "at least one asset",
		},
		{
			name:    "no output dir",
			config:  export.ExportConfig{Name: "my-plugin", Assets: []export.AssetRef{{Type: nd.AssetSkill, Name: "foo", Path: "/src/foo"}}},
			wantErr: "output directory is required",
		},
		{
			name:    "asset with empty path",
			config:  export.ExportConfig{Name: "my-plugin", OutputDir: "/tmp/out", Assets: []export.AssetRef{{Type: nd.AssetSkill, Name: "foo"}}},
			wantErr: "path is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMarketplaceConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  export.MarketplaceConfig
		wantErr string
	}{
		{
			name:    "valid minimal",
			config:  export.MarketplaceConfig{Name: "my-marketplace", Owner: export.Author{Name: "Author"}, Plugins: []export.PluginEntry{{Name: "p1", Source: "./p1"}}, OutputDir: "/tmp/out"},
			wantErr: "",
		},
		{
			name:    "empty name",
			config:  export.MarketplaceConfig{Owner: export.Author{Name: "Author"}, Plugins: []export.PluginEntry{{Name: "p1", Source: "./p1"}}, OutputDir: "/tmp/out"},
			wantErr: "name is required",
		},
		{
			name:    "not kebab-case",
			config:  export.MarketplaceConfig{Name: "My Marketplace", Owner: export.Author{Name: "Author"}, Plugins: []export.PluginEntry{{Name: "p1", Source: "./p1"}}, OutputDir: "/tmp/out"},
			wantErr: "kebab-case",
		},
		{
			name:    "empty owner name",
			config:  export.MarketplaceConfig{Name: "my-marketplace", Plugins: []export.PluginEntry{{Name: "p1", Source: "./p1"}}, OutputDir: "/tmp/out"},
			wantErr: "owner name is required",
		},
		{
			name:    "no plugins",
			config:  export.MarketplaceConfig{Name: "my-marketplace", Owner: export.Author{Name: "Author"}, OutputDir: "/tmp/out"},
			wantErr: "at least one plugin",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateAssetName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{name: "valid simple", input: "foo", wantErr: ""},
		{name: "valid with extension", input: "my-agent.md", wantErr: ""},
		{name: "valid kebab-case", input: "go-conventions", wantErr: ""},
		{name: "empty name", input: "", wantErr: "cannot be empty"},
		{name: "path traversal dotdot", input: "..", wantErr: "path traversal"},
		{name: "path traversal embedded", input: "foo..bar", wantErr: "path traversal"},
		{name: "forward slash", input: "foo/bar", wantErr: "path separators"},
		{name: "backslash", input: "foo\\bar", wantErr: "path separators"},
		{name: "bare dotdot", input: "..", wantErr: "path traversal"},
		{name: "bare dot", input: ".", wantErr: "path traversal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := export.ValidateAssetName(tt.input)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidateOutputDir(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr string
	}{
		{name: "valid absolute", dir: "/tmp/my-plugin", wantErr: ""},
		{name: "valid relative", dir: "./my-plugin", wantErr: ""},
		{name: "valid nested", dir: "output/my-plugin", wantErr: ""},
		{name: "filesystem root", dir: "/", wantErr: "not a safe target"},
		{name: "current dir dot", dir: ".", wantErr: "not a safe target"},
		{name: "parent dir dotdot", dir: "..", wantErr: "not a safe target"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := export.ValidateOutputDir(tt.dir)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestExportConfigValidate_PathTraversal(t *testing.T) {
	tests := []struct {
		name    string
		asset   string
		wantErr string
	}{
		{name: "slashes", asset: "../../etc/passwd", wantErr: "path separators"},
		{name: "dotdot only", asset: "..", wantErr: "path traversal"},
		{name: "embedded dotdot", asset: "foo..bar", wantErr: "path traversal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := export.ExportConfig{
				Name:      "my-plugin",
				OutputDir: "/tmp/out",
				Assets: []export.AssetRef{
					{Type: "skills", Name: tt.asset, Path: "/src/foo"},
				},
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestExportConfigValidate_UnsafeOutputDir(t *testing.T) {
	cfg := export.ExportConfig{
		Name:      "my-plugin",
		OutputDir: "/",
		Assets: []export.AssetRef{
			{Type: "skills", Name: "foo", Path: "/src/foo"},
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for filesystem root as output dir")
	}
	if !strings.Contains(err.Error(), "not a safe target") {
		t.Fatalf("error %q does not mention safety", err.Error())
	}
}

func TestMarketplaceConfigValidate_PluginNameTraversal(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		wantErr    string
	}{
		{name: "slashes", pluginName: "../../malicious", wantErr: "path separators"},
		{name: "dotdot only", pluginName: "..", wantErr: "path traversal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := export.MarketplaceConfig{
				Name:      "my-marketplace",
				Owner:     export.Author{Name: "Author"},
				OutputDir: "/tmp/out",
				Plugins: []export.PluginEntry{
					{Name: tt.pluginName, Source: "./p1"},
				},
			}
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestIsKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"my-plugin", true},
		{"plugin", true},
		{"my-great-plugin", true},
		{"123", true},
		{"plugin-123", true},
		{"MY-PLUGIN", false},
		{"My Plugin", false},
		{"my_plugin", false},
		{"my plugin", false},
		{"", false},
		{"-leading-dash", false},
		{"trailing-dash-", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := export.IsKebabCase(tt.input)
			if got != tt.want {
				t.Fatalf("IsKebabCase(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
