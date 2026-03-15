package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/larah/nd/internal/output"
)

// printJSON marshals a JSONResponse envelope to the writer.
func printJSON(w io.Writer, data interface{}, dryRun bool) error {
	resp := output.JSONResponse{
		Status: "ok",
		Data:   data,
	}
	if dryRun {
		resp.DryRun = true
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// printJSONError marshals a JSON error response to the writer.
func printJSONError(w io.Writer, errs []output.JSONError) error {
	status := "error"
	if len(errs) > 1 {
		status = "partial"
	}
	resp := output.JSONResponse{
		Status: status,
		Errors: errs,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// printHuman writes formatted output to the writer.
func printHuman(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...)
}

// confirm prompts for yes/no confirmation.
// Returns true immediately if yesFlag is set.
// Returns an error if stdin is not a terminal (piped input).
func confirm(r io.Reader, w io.Writer, prompt string, yesFlag bool) (bool, error) {
	if yesFlag {
		return true, nil
	}
	if !isTerminal() {
		return false, fmt.Errorf("confirmation required but stdin is not a terminal (use --yes to skip)")
	}

	fmt.Fprintf(w, "%s [y/N] ", prompt)
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false, nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}

// promptChoice presents numbered choices and returns the selected option.
// Returns an error if stdin is not a terminal.
func promptChoice(r io.Reader, w io.Writer, prompt string, choices []string) (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("interactive choice required but stdin is not a terminal")
	}

	fmt.Fprintln(w, prompt)
	for i, c := range choices {
		fmt.Fprintf(w, "  %d) %s\n", i+1, c)
	}
	fmt.Fprintf(w, "Choice [1-%d]: ", len(choices))

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input")
	}
	n, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || n < 1 || n > len(choices) {
		return "", fmt.Errorf("invalid choice: %s", scanner.Text())
	}
	return choices[n-1], nil
}

// isTerminal checks if stdin is a terminal.
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
