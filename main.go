package main

import (
	"os"

	"github.com/larah/nd/cmd"
)

func main() {
	code := cmd.Execute()
	os.Exit(code)
}
