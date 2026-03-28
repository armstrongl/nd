// Command gendocs generates Markdown command reference from Cobra definitions.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/armstrongl/nd/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	outDir := "docs/reference"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cmd.NewRootCmd(&cmd.App{})
	root.DisableAutoGenTag = true

	if err := generateCommandDocs(root, outDir); err != nil {
		log.Fatal(err)
	}
}

// generateCommandDocs generates Hugo-compatible Markdown reference pages for
// cmd and all its subcommands, writing them to outDir.
// homePathRe matches absolute home directory prefixes like /Users/foo/ or /home/foo/.
var homePathRe = regexp.MustCompile(`^/(Users|home)/[^/]+/`)

func generateCommandDocs(root *cobra.Command, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Sanitize config flag default to use portable tilde path.
	if f := root.PersistentFlags().Lookup("config"); f != nil {
		f.DefValue = homePathRe.ReplaceAllString(f.DefValue, "~/")
	}

	cmds := allCommands(root)

	// Build sorted names for stable weight assignment.
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.CommandPath())
	}
	sort.Strings(names)

	nameIndex := make(map[string]int, len(names))
	for i, n := range names {
		nameIndex[n] = i
	}

	rootName := root.Name()
	linkHandler := func(s string) string { return s }

	for _, c := range cmds {
		var buf bytes.Buffer
		if err := doc.GenMarkdownCustom(c, &buf, linkHandler); err != nil {
			return fmt.Errorf("generating %s: %w", c.CommandPath(), err)
		}

		body := buf.String()

		// Strip the H2 title line (Hextra renders front matter title as H1).
		lines := strings.SplitAfter(body, "\n")
		var out []string
		skippedTitle := false
		for _, line := range lines {
			if !skippedTitle && strings.HasPrefix(line, "## ") {
				skippedTitle = true
				continue
			}
			// Promote H3 → H2.
			if strings.HasPrefix(line, "### ") {
				line = strings.Replace(line, "### ", "## ", 1)
			}
			out = append(out, line)
		}

		weight := 1
		if c.CommandPath() != rootName {
			weight = (nameIndex[c.CommandPath()] + 1) * 10
		}

		frontMatter := fmt.Sprintf("---\ntitle: %q\nweight: %d\n---\n\n", c.CommandPath(), weight)
		content := frontMatter + strings.Join(out, "")

		filename := strings.ReplaceAll(c.CommandPath(), " ", "_") + ".md"
		path := filepath.Join(outDir, filename)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

// allCommands returns cmd and all its descendants, depth-first.
func allCommands(cmd *cobra.Command) []*cobra.Command {
	result := []*cobra.Command{cmd}
	for _, c := range cmd.Commands() {
		result = append(result, allCommands(c)...)
	}
	return result
}
