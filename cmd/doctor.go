package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/armstrongl/nd/internal/config"
	"github.com/armstrongl/nd/internal/doctor"
	"github.com/armstrongl/nd/internal/nd"
	"github.com/armstrongl/nd/internal/sourcemanager"
	"github.com/spf13/cobra"
)

func newDoctorCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check nd configuration and deployment health",
		Example: `  # Run a full health check
  nd doctor

  # Output as JSON for CI
  nd doctor --json`,
		Annotations: map[string]string{
			"docs.guides": "getting-started,troubleshooting",
		},
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			report := doctor.Report{}

			// 1. Config check
			sm, smErr := app.SourceManager()
			if smErr != nil {
				report.Config.GlobalValid = false
				report.Config.Errors = []config.ValidationError{
					{Field: "config", Message: smErr.Error()},
				}
				report.Summary.Fail++
			} else {
				cfg := sm.Config()
				errs := cfg.Validate()
				report.Config.GlobalValid = len(errs) == 0
				report.Config.Errors = errs
				if len(errs) > 0 {
					report.Summary.Fail++
				} else {
					report.Summary.Pass++
				}
			}

			// 2. Source checks
			if sm != nil {
				for _, s := range sm.Sources() {
					sc := doctor.SourceCheck{
						SourceID: s.ID,
					}
					result := sourcemanager.ScanSource(s.ID, s.Path)
					sc.AssetCount = len(result.Assets)
					if info, err := os.Stat(s.Path); err != nil || !info.IsDir() {
						sc.Available = false
						sc.Detail = fmt.Sprintf("path %q is not accessible", s.Path)
						report.Summary.Fail++
					} else {
						sc.Available = true
						report.Summary.Pass++
					}
					report.Sources = append(report.Sources, sc)
				}
			}

			// 3. Deployment health
			eng, engErr := app.DeployEngine()
			if engErr == nil {
				checks, err := eng.Check()
				if err == nil {
					report.Deployments = checks
					for _, hc := range checks {
						report.Summary.Warn++
						_ = hc
					}
					if len(checks) == 0 {
						report.Summary.Pass++
					}
				}
			}

			// 4. Agent checks
			reg, regErr := app.AgentRegistry()
			if regErr == nil {
				for _, ag := range reg.All() {
					ac := doctor.AgentCheck{
						AgentName: ag.Name,
						Detected:  ag.Detected || ag.InPath,
						GlobalDir: ag.GlobalDir,
					}
					if info, err := os.Stat(ag.GlobalDir); err == nil && info.IsDir() {
						ac.GlobalOK = true
						report.Summary.Pass++
					} else {
						ac.GlobalOK = false
						ac.Detail = fmt.Sprintf("global dir %q not found", ag.GlobalDir)
						report.Summary.Warn++
					}
					report.Agents = append(report.Agents, ac)
				}
			}

			// 5. Git check
			gitOut, err := exec.Command("git", "--version").Output()
			if err == nil {
				report.Git.Available = true
				report.Git.Version = strings.TrimSpace(string(gitOut))
				report.Summary.Pass++
			} else {
				report.Git.Available = false
				report.Git.Detail = "git not found in PATH"
				report.Summary.Warn++
			}

			if app.JSON {
				return printJSON(w, report, app.DryRun)
			}

			// Human output
			printCheck(w, report.Config.GlobalValid, "Config", "")
			for _, e := range report.Config.Errors {
				printHuman(w, "    %s\n", e.Error())
			}

			for _, sc := range report.Sources {
				printCheck(w, sc.Available, fmt.Sprintf("Source: %s", sc.SourceID),
					fmt.Sprintf("%d assets", sc.AssetCount))
			}

			if len(report.Deployments) == 0 {
				printCheck(w, true, "Deployments", "all healthy")
			} else {
				printCheck(w, false, "Deployments", fmt.Sprintf("%d issues", len(report.Deployments)))
				for _, hc := range report.Deployments {
					printHuman(w, "    %s/%s: %s\n", hc.Deployment.AssetType, hc.Deployment.AssetName, hc.Detail)
				}
			}

			for _, ac := range report.Agents {
				label := fmt.Sprintf("Agent: %s", ac.AgentName)
				if ac.Detected {
					printCheck(w, ac.GlobalOK, label, ac.GlobalDir)
				} else {
					printHuman(w, "  ? %s (not detected)\n", label)
				}
			}

			printCheck(w, report.Git.Available, "Git", report.Git.Version)

			printHuman(w, "\n%d pass, %d warn, %d fail\n",
				report.Summary.Pass, report.Summary.Warn, report.Summary.Fail)

			if report.Summary.Fail > 0 {
				return withExitCode(nd.ExitError, fmt.Errorf("%d checks failed", report.Summary.Fail))
			}
			return nil
		},
	}
}

func printCheck(w interface{ Write([]byte) (int, error) }, ok bool, label, detail string) {
	mark := "✓"
	if !ok {
		mark = "✗"
	}
	if detail != "" {
		fmt.Fprintf(w, "  %s %s: %s\n", mark, label, detail)
	} else {
		fmt.Fprintf(w, "  %s %s\n", mark, label)
	}
}
