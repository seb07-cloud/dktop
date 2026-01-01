package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/seb07-cloud/dktop/internal/theme"
)

type HelpBar struct {
	width int
}

func NewHelpBar() *HelpBar {
	return &HelpBar{}
}

func (h *HelpBar) SetWidth(width int) {
	h.width = width
}

func (h *HelpBar) View(panel Panel) string {
	var keys []struct {
		key  string
		desc string
	}

	// Common keys
	common := []struct {
		key  string
		desc string
	}{
		{"Tab", "panel"},
		{"j/k", "nav"},
		{"/", "filter"},
		{"q", "quit"},
	}

	// Panel-specific keys
	switch panel {
	case PanelContainers:
		keys = []struct {
			key  string
			desc string
		}{
			{"s", "start"},
			{"x", "stop"},
			{"r", "restart"},
			{"d", "delete"},
			{"a", "autostart"},
			{"Enter", "logs"},
		}
	case PanelImages:
		keys = []struct {
			key  string
			desc string
		}{
			{"p", "pull"},
			{"d", "delete"},
		}
	case PanelLogs:
		keys = []struct {
			key  string
			desc string
		}{
			{"j/k", "scroll"},
			{"G", "bottom"},
			{"Esc", "back"},
		}
	default:
		keys = []struct {
			key  string
			desc string
		}{}
	}

	keys = append(keys, common...)

	var parts []string
	for _, k := range keys {
		keyStyled := theme.HelpKeyStyle.Render(k.key)
		descStyled := theme.HelpStyle.Render(":" + k.desc)
		parts = append(parts, keyStyled+descStyled)
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, spaceBetween(parts, h.width)...)
}

func spaceBetween(items []string, width int) []string {
	if len(items) == 0 {
		return items
	}

	totalLen := 0
	for _, item := range items {
		totalLen += lipgloss.Width(item)
	}

	if totalLen >= width {
		return items
	}

	spacing := (width - totalLen) / len(items)
	if spacing < 1 {
		spacing = 1
	}

	spacer := lipgloss.NewStyle().Width(spacing).Render("")
	result := make([]string, 0, len(items)*2-1)
	for i, item := range items {
		result = append(result, item)
		if i < len(items)-1 {
			result = append(result, spacer)
		}
	}

	return result
}
