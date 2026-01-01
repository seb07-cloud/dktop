package ui

import (
	"fmt"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/charmbracelet/lipgloss"
	"github.com/seb/dktop/internal/docker"
	"github.com/seb/dktop/internal/theme"
)

const sparklineHeight = 4

type StatsPanel struct {
	width        int
	height       int
	stats        *docker.SystemStats
	active       bool
	cpuSparkline sparkline.Model
	memSparkline sparkline.Model
	initialized  bool
}

func NewStatsPanel() *StatsPanel {
	return &StatsPanel{}
}

func (p *StatsPanel) SetSize(width, height int) {
	p.width = width
	p.height = height

	// Initialize or resize sparklines
	graphWidth := width - 8
	if graphWidth < 10 {
		graphWidth = 10
	}

	if !p.initialized {
		p.cpuSparkline = sparkline.New(graphWidth, sparklineHeight)
		p.cpuSparkline.Style = lipgloss.NewStyle().Foreground(theme.Cyan)
		p.cpuSparkline.SetMax(100)

		p.memSparkline = sparkline.New(graphWidth, sparklineHeight)
		p.memSparkline.Style = lipgloss.NewStyle().Foreground(theme.Purple)
		p.memSparkline.SetMax(100)

		p.initialized = true
	} else {
		p.cpuSparkline.Resize(graphWidth, sparklineHeight)
		p.memSparkline.Resize(graphWidth, sparklineHeight)
	}
}

func (p *StatsPanel) SetActive(active bool) {
	p.active = active
}

func (p *StatsPanel) Update(stats *docker.SystemStats) {
	p.stats = stats

	if stats != nil && p.initialized {
		// Push CPU data
		p.cpuSparkline.Push(stats.CPUUsage)

		// Push memory percentage
		memPerc := 0.0
		if stats.MemoryLimit > 0 {
			memPerc = float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100
		}
		p.memSparkline.Push(memPerc)
	}
}

func (p *StatsPanel) View() string {
	style := theme.PanelStyle
	if p.active {
		style = theme.ActivePanelStyle
	}

	title := theme.TitleStyle.Render(" Docker Stats ")

	if p.stats == nil {
		content := theme.InactiveStyle.Render("Loading...")
		return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n" + content)
	}

	// Container counts
	containersLine := fmt.Sprintf("Containers: %s %s %s %s",
		theme.HighlightStyle.Render(fmt.Sprintf("%d", p.stats.Containers)),
		theme.RunningStyle.Render(fmt.Sprintf("(%d running)", p.stats.ContainersRunning)),
		theme.PausedStyle.Render(fmt.Sprintf("(%d paused)", p.stats.ContainersPaused)),
		theme.StoppedStyle.Render(fmt.Sprintf("(%d stopped)", p.stats.ContainersStopped)),
	)

	// Images count
	imagesLine := fmt.Sprintf("Images: %s",
		theme.HighlightStyle.Render(fmt.Sprintf("%d", p.stats.Images)),
	)

	// Memory stats
	memUsed := docker.FormatBytes(p.stats.MemoryUsage)
	memTotal := docker.FormatBytes(p.stats.MemoryLimit)
	memPerc := 0.0
	if p.stats.MemoryLimit > 0 {
		memPerc = float64(p.stats.MemoryUsage) / float64(p.stats.MemoryLimit) * 100
	}

	// CPU header with label and value (cyan to match graph)
	cpuLabelStyle := lipgloss.NewStyle().Foreground(theme.Cyan).Bold(true)
	cpuLabel := cpuLabelStyle.Render("CPU:")
	cpuValue := theme.GetUsageStyle(p.stats.CPUUsage).Render(fmt.Sprintf(" %.1f%%", p.stats.CPUUsage))
	cpuHeader := cpuLabel + cpuValue

	// Memory header with label and value (purple to match graph)
	memLabelStyle := lipgloss.NewStyle().Foreground(theme.Purple).Bold(true)
	memLabel := memLabelStyle.Render("MEM:")
	memValue := theme.GetUsageStyle(memPerc).Render(fmt.Sprintf(" %s/%s", memUsed, memTotal))
	memHeader := memLabel + memValue

	// Draw sparklines using column style (more visible than braille)
	p.cpuSparkline.Draw()
	p.memSparkline.Draw()

	content := lipgloss.JoinVertical(lipgloss.Left,
		containersLine,
		imagesLine,
		"",
		cpuHeader,
		p.cpuSparkline.View(),
		"",
		memHeader,
		p.memSparkline.View(),
	)

	return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n" + content)
}
