// Command gendocs generates Markdown command reference from Cobra definitions.
package main

import (
	"log"
	"os"

	"github.com/armstrongl/nd/cmd"
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

	rootCmd := cmd.NewRootCmd(&cmd.App{})
	rootCmd.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(rootCmd, outDir); err != nil {
		log.Fatal(err)
	}
}
