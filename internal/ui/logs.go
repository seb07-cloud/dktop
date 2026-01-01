package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/seb07-cloud/dktop/internal/theme"
)

type LogsPanel struct {
	width         int
	height        int
	logs          []string
	containerName string
	active        bool
	offset        int
	autoScroll    bool
}

func NewLogsPanel() *LogsPanel {
	return &LogsPanel{
		autoScroll: true,
	}
}

func (p *LogsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

func (p *LogsPanel) SetActive(active bool) {
	p.active = active
}

func (p *LogsPanel) SetContainerName(name string) {
	if p.containerName != name {
		p.containerName = name
		p.logs = nil
		p.offset = 0
		p.autoScroll = true
	}
}

func (p *LogsPanel) Update(logs string) {
	lines := strings.Split(logs, "\n")
	p.logs = lines

	// Auto-scroll to bottom if enabled
	if p.autoScroll {
		visibleLines := p.height - 4
		if len(p.logs) > visibleLines {
			p.offset = len(p.logs) - visibleLines
		}
	}
}

func (p *LogsPanel) AppendLog(line string) {
	p.logs = append(p.logs, line)

	// Keep only last 1000 lines
	if len(p.logs) > 1000 {
		p.logs = p.logs[len(p.logs)-1000:]
	}

	// Auto-scroll to bottom if enabled
	if p.autoScroll {
		visibleLines := p.height - 4
		if len(p.logs) > visibleLines {
			p.offset = len(p.logs) - visibleLines
		}
	}
}

func (p *LogsPanel) ScrollUp() {
	if p.offset > 0 {
		p.offset--
		p.autoScroll = false
	}
}

func (p *LogsPanel) ScrollDown() {
	visibleLines := p.height - 4
	maxOffset := len(p.logs) - visibleLines
	if maxOffset < 0 {
		maxOffset = 0
	}
	if p.offset < maxOffset {
		p.offset++
	}
	// Re-enable auto-scroll if at bottom
	if p.offset >= maxOffset {
		p.autoScroll = true
	}
}

func (p *LogsPanel) ScrollToBottom() {
	visibleLines := p.height - 4
	if len(p.logs) > visibleLines {
		p.offset = len(p.logs) - visibleLines
	}
	p.autoScroll = true
}

func (p *LogsPanel) View() string {
	style := theme.PanelStyle
	if p.active {
		style = theme.ActivePanelStyle
	}

	title := theme.TitleStyle.Render(" Logs ")
	if p.containerName != "" {
		title += theme.InactiveStyle.Render(" [" + p.containerName + "]")
	}

	if len(p.logs) == 0 || p.containerName == "" {
		var msg string
		if p.containerName == "" {
			msg = "Select a container to view logs"
		} else {
			msg = "No logs available"
		}
		content := theme.InactiveStyle.Render(msg)
		return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n\n" + content)
	}

	visibleLines := p.height - 4
	if visibleLines < 1 {
		visibleLines = 1
	}

	var rows []string
	maxWidth := p.width - 6

	for i := p.offset; i < len(p.logs) && i < p.offset+visibleLines; i++ {
		line := p.logs[i]

		// Truncate long lines
		if len(line) > maxWidth {
			line = line[:maxWidth-3] + "..."
		}

		// Color timestamps differently
		if len(line) > 0 {
			// Try to find timestamp (usually first part before space)
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 && (strings.Contains(parts[0], "T") || strings.Contains(parts[0], ":")) {
				line = theme.InactiveStyle.Render(parts[0]) + " " + parts[1]
			}
		}

		rows = append(rows, line)
	}

	// Scroll indicator
	scrollInfo := ""
	if !p.autoScroll {
		scrollInfo = theme.InactiveStyle.Render(" (scroll locked - press G to unlock)")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return style.Width(p.width - 2).Height(p.height - 2).Render(title + scrollInfo + "\n" + content)
}
