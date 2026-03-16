package main

import (
	"os"

	"github.com/armstrongl/nd/cmd"
)

func main() {
	code := cmd.Execute()
	os.Exit(code)
}
