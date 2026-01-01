package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/seb07-cloud/dktop/internal/docker"
	"github.com/seb07-cloud/dktop/internal/theme"
)

type ContainersPanel struct {
	width      int
	height     int
	containers []docker.ContainerInfo
	selected   int
	offset     int
	active     bool
	filter     string
}

func NewContainersPanel() *ContainersPanel {
	return &ContainersPanel{}
}

func (p *ContainersPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

func (p *ContainersPanel) SetActive(active bool) {
	p.active = active
}

func (p *ContainersPanel) Update(containers []docker.ContainerInfo) {
	p.containers = containers
	// Ensure selection is valid
	if p.selected >= len(p.containers) {
		p.selected = len(p.containers) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
}

func (p *ContainersPanel) SetFilter(filter string) {
	p.filter = filter
	p.selected = 0
	p.offset = 0
}

func (p *ContainersPanel) GetFiltered() []docker.ContainerInfo {
	if p.filter == "" {
		return p.containers
	}

	var filtered []docker.ContainerInfo
	filterLower := strings.ToLower(p.filter)
	for _, c := range p.containers {
		if strings.Contains(strings.ToLower(c.Name), filterLower) ||
			strings.Contains(strings.ToLower(c.Image), filterLower) ||
			strings.Contains(strings.ToLower(c.ID), filterLower) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (p *ContainersPanel) MoveUp() {
	if p.selected > 0 {
		p.selected--
		if p.selected < p.offset {
			p.offset = p.selected
		}
	}
}

func (p *ContainersPanel) MoveDown() {
	filtered := p.GetFiltered()
	if p.selected < len(filtered)-1 {
		p.selected++
		visibleRows := p.height - 5 // Account for border, title, header
		if p.selected >= p.offset+visibleRows {
			p.offset = p.selected - visibleRows + 1
		}
	}
}

func (p *ContainersPanel) GetSelected() *docker.ContainerInfo {
	filtered := p.GetFiltered()
	if p.selected >= 0 && p.selected < len(filtered) {
		return &filtered[p.selected]
	}
	return nil
}

func (p *ContainersPanel) View() string {
	style := theme.PanelStyle
	if p.active {
		style = theme.ActivePanelStyle
	}

	title := theme.TitleStyle.Render(" Containers ")
	if p.filter != "" {
		title += theme.InactiveStyle.Render(fmt.Sprintf(" [filter: %s]", p.filter))
	}

	filtered := p.GetFiltered()

	if len(filtered) == 0 {
		content := theme.InactiveStyle.Render("No containers found")
		return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n\n" + content)
	}

	// Calculate column widths - use proportional widths based on panel width
	availableWidth := p.width - 10 // Account for borders and padding

	nameW := availableWidth * 18 / 100   // 18%
	statusW := availableWidth * 12 / 100 // 12%
	cpuW := 8                             // Fixed width for CPU %
	memW := 10                            // Fixed width for memory
	portsW := availableWidth * 22 / 100  // 22%
	imageW := availableWidth - nameW - statusW - cpuW - memW - portsW - 6 // Remaining space, -6 for separators

	// Minimum widths
	if nameW < 12 {
		nameW = 12
	}
	if statusW < 10 {
		statusW = 10
	}
	if portsW < 10 {
		portsW = 10
	}
	if imageW < 10 {
		imageW = 10
	}

	// Header
	header := fmt.Sprintf("%-*s %-*s %*s %*s %-*s %-*s",
		nameW, "NAME",
		statusW, "STATUS",
		cpuW, "CPU",
		memW, "MEM",
		portsW, "PORTS",
		imageW, "IMAGE",
	)
	headerStyled := theme.HighlightStyle.Render(header)

	// Rows
	visibleRows := p.height - 5
	if visibleRows < 1 {
		visibleRows = 1
	}

	var rows []string
	for i := p.offset; i < len(filtered) && i < p.offset+visibleRows; i++ {
		c := filtered[i]
		isSelected := i == p.selected

		name := truncate(c.Name, nameW)
		status := truncate(c.Status, statusW)
		cpu := fmt.Sprintf("%5.1f%%", c.CPUPerc)
		mem := fmt.Sprintf("%*s", memW, docker.FormatBytesShort(c.MemUsage))
		ports := truncate(c.Ports, portsW)
		img := truncate(c.Image, imageW)

		var row string
		if isSelected {
			// For selected row, use plain text and apply selection style to entire row
			autostart := " "
			if c.Autostart {
				autostart = "A"
			}
			row = fmt.Sprintf("%s%-*s %-*s %*s %s %-*s %-*s",
				autostart,
				nameW-1, name,
				statusW, status,
				cpuW, cpu,
				mem,
				portsW, ports,
				imageW, img,
			)
			row = theme.SelectedStyle.Width(p.width - 4).Render(row)
		} else {
			// For non-selected rows, apply individual colors with fixed widths
			var statusStyled string
			switch c.State {
			case "running":
				statusStyled = theme.RunningStyle.Width(statusW).Render(status)
			case "paused":
				statusStyled = theme.PausedStyle.Width(statusW).Render(status)
			default:
				statusStyled = theme.StoppedStyle.Width(statusW).Render(status)
			}

			cpuStyled := theme.GetUsageStyle(c.CPUPerc).Width(cpuW).Render(cpu)
			memStyled := theme.GetUsageStyle(c.MemPerc).Render(mem)

			autostart := " "
			if c.Autostart {
				autostart = theme.HighlightStyle.Render("A")
			}

			nameStyled := lipgloss.NewStyle().Width(nameW - 1).Render(name)
			portsStyled := lipgloss.NewStyle().Width(portsW).Render(ports)
			imgStyled := lipgloss.NewStyle().Width(imageW).Render(img)

			row = autostart + nameStyled + " " + statusStyled + " " + cpuStyled + " " + memStyled + " " + portsStyled + " " + imgStyled
		}

		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{headerStyled, ""}, rows...)...)

	return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n" + content)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
