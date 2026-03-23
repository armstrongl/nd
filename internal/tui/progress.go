package tui

import (
	"fmt"

	"charm.land/bubbles/v2/progress"
)

// progressMsg is sent during bulk operations to update the progress display.
type progressMsg struct {
	completed int
	total     int
	name      string
}

// progressBar wraps a bubbles progress bar with a counter and current item name.
type progressBar struct {
	bar       progress.Model
	completed int
	total     int
	name      string
}

func newProgressBar(width int) progressBar {
	bar := progress.New(
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return progressBar{bar: bar}
}

// Update processes a progressMsg and returns the updated bar.
func (p progressBar) Update(msg progressMsg) progressBar {
	p.completed = msg.completed
	p.total = msg.total
	p.name = msg.name
	return p
}

// View renders the progress bar with counter and item name.
func (p progressBar) View(s Styles) string {
	if p.total == 0 {
		return ""
	}

	percent := float64(p.completed) / float64(p.total)
	bar := p.bar.ViewAs(percent)
	counter := fmt.Sprintf("%d/%d", p.completed, p.total)

	result := fmt.Sprintf("  %s  %s", bar, s.Subtle.Render(counter))
	if p.name != "" {
		result += fmt.Sprintf("\n  %s", s.Subtle.Render(p.name))
	}
	return result
}
