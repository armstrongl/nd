package cmd

import (
	"os"

	"github.com/armstrongl/nd/internal/nd"
	"github.com/spf13/cobra"
)

// Labels shown by the interactive deploy-scope prompt. The copy states the
// default agent's destination directories (see internal/agent/registry.go):
// global deploys to ~/.claude/, project deploys to .claude/ in the project root.
const (
	scopeChoiceGlobal  = "Global (system-wide, ~/.claude/)"
	scopeChoiceProject = "Project (this project, .claude/)"
)

// deployScopePref remembers the scope the user picked interactively for the
// duration of a single process. A nil value means "not chosen yet". It resets
// naturally on process exit; tests clear it with resetDeployScopePref.
var deployScopePref *nd.Scope

// resetDeployScopePref clears the remembered session scope preference. Intended
// for tests that need to isolate cases within one process.
func resetDeployScopePref() {
	deployScopePref = nil
}

// resolveDeployScope decides the deployment scope for a deploy invocation,
// prompting interactively only when it is safe and useful to do so.
//
// Resolution order:
//  1. An explicit --scope flag always wins, with no prompt.
//  2. A choice already made earlier in this process is reused (no re-prompt).
//  3. Non-interactive invocations (--json or a non-terminal stdin) keep the
//     current default with no prompt and no error, so scripted callers are
//     unaffected.
//  4. Outside a project (no .git/.claude found walking up from cwd) global is
//     the only sensible scope, so it is used without prompting.
//  5. Otherwise the user is asked once to choose global vs project; the answer
//     is remembered for the rest of the process.
func resolveDeployScope(cmd *cobra.Command, app *App) (nd.Scope, error) {
	// 1. Explicit flag wins.
	if cmd.Flags().Changed("scope") {
		return app.Scope, nil
	}

	// 2. Reuse an earlier choice from this process.
	if deployScopePref != nil {
		return *deployScopePref, nil
	}

	// 3. Non-interactive: keep the current default, no prompt, no error.
	if app.JSON || !isTerminal() {
		return app.Scope, nil
	}

	// 4. Outside a project, project scope is not meaningful: use global.
	cwd, err := os.Getwd()
	if err != nil {
		return app.Scope, nil
	}
	if _, err := nd.FindProjectRoot(cwd); err != nil {
		scope := nd.ScopeGlobal
		deployScopePref = &scope
		return scope, nil
	}

	// 5. Prompt once and remember the answer.
	choice, err := promptChoice(cmd.InOrStdin(), cmd.OutOrStdout(), "Deploy scope:",
		[]string{scopeChoiceGlobal, scopeChoiceProject})
	if err != nil {
		return app.Scope, err
	}
	scope := nd.ScopeGlobal
	if choice == scopeChoiceProject {
		scope = nd.ScopeProject
	}
	deployScopePref = &scope
	return scope, nil
}
