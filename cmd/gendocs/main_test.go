package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestCommandTree builds a minimal Cobra command tree for testing.
func newTestCommandTree() *cobra.Command {
	root := &cobra.Command{
		Use:   "testcli",
		Short: "A test CLI tool",
		Long:  "testcli manages widgets for testing purposes.",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	root.PersistentFlags().String("config", "/Users/testuser/.config/testcli/config.yaml", "path to config file")
	root.DisableAutoGenTag = true

	noop := func(cmd *cobra.Command, args []string) {}

	deploy := &cobra.Command{
		Use:   "deploy <widget> [flags]",
		Short: "Deploy widgets",
		Long:  "Deploy one or more widgets by creating symlinks.",
		Annotations: map[string]string{
			"docs.related": "testcli remove,testcli profile",
		},
		Run: noop,
	}
	deploy.Flags().Bool("force", false, "force deployment")
	root.AddCommand(deploy)

	remove := &cobra.Command{
		Use:   "remove <widget>",
		Short: "Remove deployed widgets",
		Long:  "Remove one or more deployed widgets.",
		Annotations: map[string]string{
			"docs.related": "testcli deploy",
		},
		Run: noop,
	}
	root.AddCommand(remove)

	profile := &cobra.Command{
		Use:   "profile",
		Short: "Manage profiles",
		Long:  "Manage deployment profiles for the CLI.",
		Run:   noop,
	}
	root.AddCommand(profile)

	profileCreate := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new profile",
		Long:  "Create a new deployment profile with given name.",
		Run:   noop,
	}
	profile.AddCommand(profileCreate)

	return root
}

func TestGenerateCommandDocs_FrontMatter(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	// Check that all generated files have valid YAML front matter with title field.
	files := []struct {
		filename string
		title    string
	}{
		{"testcli.md", "testcli"},
		{"testcli_deploy.md", "testcli deploy"},
		{"testcli_remove.md", "testcli remove"},
		{"testcli_profile.md", "testcli profile"},
		{"testcli_profile_create.md", "testcli profile create"},
	}
	for _, f := range files {
		content, err := os.ReadFile(filepath.Join(outDir, f.filename))
		if err != nil {
			t.Fatalf("failed to read %s: %v", f.filename, err)
		}
		text := string(content)

		// Must start with front matter delimiter.
		if !strings.HasPrefix(text, "---\n") {
			t.Errorf("%s: expected to start with '---\\n', got first 40 chars: %q", f.filename, text[:min(40, len(text))])
		}

		// Must have closing front matter delimiter.
		closingIdx := strings.Index(text[4:], "\n---\n")
		if closingIdx == -1 {
			t.Errorf("%s: missing closing front matter delimiter", f.filename)
		}

		// Must have correct title in front matter.
		expectedTitle := `title: "` + f.title + `"`
		if !strings.Contains(text, expectedTitle) {
			t.Errorf("%s: expected front matter to contain %q", f.filename, expectedTitle)
		}
	}
}

func TestGenerateCommandDocs_NoH2Title(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		content, err := os.ReadFile(filepath.Join(outDir, e.Name()))
		if err != nil {
			t.Fatalf("failed to read %s: %v", e.Name(), err)
		}
		// Get body after front matter.
		text := string(content)
		body := bodyAfterFrontMatter(text)

		// The body must NOT start with "## " (the H2 title that Cobra generates).
		if strings.HasPrefix(strings.TrimLeft(body, "\n"), "## ") {
			firstLine := strings.SplitN(body, "\n", 2)[0]
			t.Errorf("%s: body should not start with H2 title, got: %q", e.Name(), firstLine)
		}
	}
}

func TestGenerateCommandDocs_H3PromotedToH2(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		content, err := os.ReadFile(filepath.Join(outDir, e.Name()))
		if err != nil {
			t.Fatalf("failed to read %s: %v", e.Name(), err)
		}
		text := string(content)

		// Must NOT contain any H3 headings (### ) — all should be promoted to H2 (## ).
		for i, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "### ") {
				t.Errorf("%s line %d: found H3 heading that should have been promoted to H2: %q", e.Name(), i+1, line)
			}
		}

		// Body SHOULD contain H2 headings (## ) for sections like Synopsis, Options, SEE ALSO.
		body := bodyAfterFrontMatter(text)
		if !strings.Contains(body, "## ") {
			t.Errorf("%s: expected H2 headings in body after H3→H2 promotion", e.Name())
		}
	}
}

func TestGenerateCommandDocs_NoHardcodedPaths(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		content, err := os.ReadFile(filepath.Join(outDir, e.Name()))
		if err != nil {
			t.Fatalf("failed to read %s: %v", e.Name(), err)
		}
		text := string(content)

		if strings.Contains(text, "/Users/") {
			t.Errorf("%s: contains hardcoded /Users/ path", e.Name())
		}
	}
}

func TestGenerateCommandDocs_SeeAlsoLinks(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	// The root command has subcommands, so its SEE ALSO section should have .md links.
	content, err := os.ReadFile(filepath.Join(outDir, "testcli.md"))
	if err != nil {
		t.Fatalf("failed to read testcli.md: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "(testcli_deploy.md)") {
		t.Error("testcli.md: expected SEE ALSO link to testcli_deploy.md")
	}
	if !strings.Contains(text, "(testcli_profile.md)") {
		t.Error("testcli.md: expected SEE ALSO link to testcli_profile.md")
	}

	// Subcommand should link back to parent with .md.
	deployContent, err := os.ReadFile(filepath.Join(outDir, "testcli_deploy.md"))
	if err != nil {
		t.Fatalf("failed to read testcli_deploy.md: %v", err)
	}
	if !strings.Contains(string(deployContent), "(testcli.md)") {
		t.Error("testcli_deploy.md: expected SEE ALSO link back to testcli.md")
	}
}

func TestGenerateCommandDocs_FileNaming(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	expectedFiles := []string{
		"testcli.md",
		"testcli_deploy.md",
		"testcli_remove.md",
		"testcli_profile.md",
		"testcli_profile_create.md",
	}

	entries, _ := os.ReadDir(outDir)
	got := make(map[string]bool)
	for _, e := range entries {
		got[e.Name()] = true
	}

	for _, want := range expectedFiles {
		if !got[want] {
			t.Errorf("expected file %s to exist", want)
		}
	}
	if len(entries) != len(expectedFiles) {
		t.Errorf("expected %d files, got %d", len(expectedFiles), len(entries))
	}
}

func TestGenerateCommandDocs_TitleMatchesCommandPath(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	tests := []struct {
		filename string
		title    string
	}{
		{"testcli.md", "testcli"},
		{"testcli_deploy.md", "testcli deploy"},
		{"testcli_remove.md", "testcli remove"},
		{"testcli_profile.md", "testcli profile"},
		{"testcli_profile_create.md", "testcli profile create"},
	}

	for _, tt := range tests {
		content, err := os.ReadFile(filepath.Join(outDir, tt.filename))
		if err != nil {
			t.Fatalf("failed to read %s: %v", tt.filename, err)
		}
		text := string(content)
		expected := `title: "` + tt.title + `"`
		if !strings.Contains(text, expected) {
			t.Errorf("%s: expected title %q, not found in front matter", tt.filename, tt.title)
		}
	}
}

func TestGenerateCommandDocs_ConfigDefaultOverridden(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()
	// Set the config flag default to a hardcoded absolute path to verify it gets overridden.
	flag := root.PersistentFlags().Lookup("config")
	if flag != nil {
		flag.DefValue = "/Users/testuser/.config/testcli/config.yaml"
	}

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	// The generated output should use ~/.config/... not the absolute path.
	content, err := os.ReadFile(filepath.Join(outDir, "testcli.md"))
	if err != nil {
		t.Fatalf("failed to read testcli.md: %v", err)
	}
	text := string(content)

	if strings.Contains(text, "/Users/testuser/") {
		t.Error("testcli.md: config flag default still contains hardcoded /Users/ path")
	}
	if !strings.Contains(text, "~/.config/") {
		t.Error("testcli.md: expected config flag default to use ~/.config/ tilde path")
	}
}

func TestGenerateCommandDocs_DocsRelated_HappyPath(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	// deploy has docs.related: "testcli remove,testcli profile"
	content, err := os.ReadFile(filepath.Join(outDir, "testcli_deploy.md"))
	if err != nil {
		t.Fatalf("failed to read testcli_deploy.md: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "## Related") {
		t.Error("testcli_deploy.md: expected ## Related section")
	}
	if !strings.Contains(text, "[testcli remove](testcli_remove.md) - Remove deployed widgets") {
		t.Error("testcli_deploy.md: expected docs.related link to testcli remove")
	}
	if !strings.Contains(text, "[testcli profile](testcli_profile.md) - Manage profiles") {
		t.Error("testcli_deploy.md: expected docs.related link to testcli profile")
	}
}

func TestGenerateCommandDocs_DocsRelated_UnknownCommand(t *testing.T) {
	outDir := t.TempDir()
	root := &cobra.Command{
		Use:   "testcli",
		Short: "A test CLI tool",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	root.DisableAutoGenTag = true

	orphan := &cobra.Command{
		Use:   "orphan",
		Short: "Orphan command",
		Annotations: map[string]string{
			"docs.related": "testcli nonexistent",
		},
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(orphan)

	// Must not crash; the unknown reference is silently skipped (with stderr warning).
	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "testcli_orphan.md"))
	if err != nil {
		t.Fatalf("failed to read testcli_orphan.md: %v", err)
	}
	text := string(content)

	// The nonexistent command must NOT appear as a link.
	if strings.Contains(text, "testcli nonexistent") {
		t.Error("testcli_orphan.md: should not contain link to nonexistent command")
	}
}

func TestGenerateCommandDocs_DocsRelated_EmptyValue(t *testing.T) {
	outDir := t.TempDir()
	root := &cobra.Command{
		Use:   "testcli",
		Short: "A test CLI tool",
		Run:   func(cmd *cobra.Command, args []string) {},
	}
	root.DisableAutoGenTag = true

	empty := &cobra.Command{
		Use:   "empty",
		Short: "Empty related",
		Annotations: map[string]string{
			"docs.related": "",
		},
		Run: func(cmd *cobra.Command, args []string) {},
	}
	root.AddCommand(empty)

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(outDir, "testcli_empty.md"))
	if err != nil {
		t.Fatalf("failed to read testcli_empty.md: %v", err)
	}
	text := string(content)

	// Empty docs.related should NOT add any extra links beyond what Cobra generates.
	body := bodyAfterFrontMatter(text)
	// Count link lines — should only have the Cobra-generated parent link.
	var linkCount int
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "- [") {
			linkCount++
		}
	}
	// The "empty" subcommand gets a Cobra-generated link back to "testcli".
	if linkCount != 1 {
		t.Errorf("testcli_empty.md: expected 1 link (Cobra parent), got %d", linkCount)
	}
}

func TestGenerateCommandDocs_DocsRelated_CombinedWithCobraLinks(t *testing.T) {
	outDir := t.TempDir()
	root := newTestCommandTree()

	if err := generateCommandDocs(root, outDir); err != nil {
		t.Fatalf("generateCommandDocs failed: %v", err)
	}

	// deploy has a Cobra-generated parent link to "testcli" AND docs.related links.
	content, err := os.ReadFile(filepath.Join(outDir, "testcli_deploy.md"))
	if err != nil {
		t.Fatalf("failed to read testcli_deploy.md: %v", err)
	}
	text := string(content)

	// Must have Cobra-generated parent link.
	if !strings.Contains(text, "(testcli.md)") {
		t.Error("testcli_deploy.md: expected Cobra-generated parent link to testcli.md")
	}
	// Must also have docs.related links.
	if !strings.Contains(text, "(testcli_remove.md)") {
		t.Error("testcli_deploy.md: expected docs.related link to testcli_remove.md")
	}
	if !strings.Contains(text, "(testcli_profile.md)") {
		t.Error("testcli_deploy.md: expected docs.related link to testcli_profile.md")
	}

	// All links should be under a single ## Related section.
	body := bodyAfterFrontMatter(text)
	relatedCount := strings.Count(body, "## Related")
	if relatedCount != 1 {
		t.Errorf("testcli_deploy.md: expected exactly 1 '## Related' heading, got %d", relatedCount)
	}
}

// bodyAfterFrontMatter returns the content after the YAML front matter block.
func bodyAfterFrontMatter(text string) string {
	if !strings.HasPrefix(text, "---\n") {
		return text
	}
	closingIdx := strings.Index(text[4:], "\n---\n")
	if closingIdx == -1 {
		return text
	}
	return text[4+closingIdx+5:]
}
