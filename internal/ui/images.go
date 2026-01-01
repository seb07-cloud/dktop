package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/seb07-cloud/dktop/internal/docker"
	"github.com/seb07-cloud/dktop/internal/theme"
)

type ImagesPanel struct {
	width    int
	height   int
	images   []docker.ImageInfo
	selected int
	offset   int
	active   bool
	filter   string
}

func NewImagesPanel() *ImagesPanel {
	return &ImagesPanel{}
}

func (p *ImagesPanel) SetSize(width, height int) {
	p.width = width
	p.height = height
}

func (p *ImagesPanel) SetActive(active bool) {
	p.active = active
}

func (p *ImagesPanel) Update(images []docker.ImageInfo) {
	p.images = images
	if p.selected >= len(p.images) {
		p.selected = len(p.images) - 1
	}
	if p.selected < 0 {
		p.selected = 0
	}
}

func (p *ImagesPanel) SetFilter(filter string) {
	p.filter = filter
	p.selected = 0
	p.offset = 0
}

func (p *ImagesPanel) GetFiltered() []docker.ImageInfo {
	if p.filter == "" {
		return p.images
	}

	var filtered []docker.ImageInfo
	filterLower := strings.ToLower(p.filter)
	for _, img := range p.images {
		for _, tag := range img.Tags {
			if strings.Contains(strings.ToLower(tag), filterLower) {
				filtered = append(filtered, img)
				break
			}
		}
		if strings.Contains(strings.ToLower(img.ID), filterLower) {
			filtered = append(filtered, img)
		}
	}
	return filtered
}

func (p *ImagesPanel) MoveUp() {
	if p.selected > 0 {
		p.selected--
		if p.selected < p.offset {
			p.offset = p.selected
		}
	}
}

func (p *ImagesPanel) MoveDown() {
	filtered := p.GetFiltered()
	if p.selected < len(filtered)-1 {
		p.selected++
		visibleRows := p.height - 5
		if p.selected >= p.offset+visibleRows {
			p.offset = p.selected - visibleRows + 1
		}
	}
}

func (p *ImagesPanel) GetSelected() *docker.ImageInfo {
	filtered := p.GetFiltered()
	if p.selected >= 0 && p.selected < len(filtered) {
		return &filtered[p.selected]
	}
	return nil
}

func (p *ImagesPanel) View() string {
	style := theme.PanelStyle
	if p.active {
		style = theme.ActivePanelStyle
	}

	title := theme.TitleStyle.Render(" Images ")
	if p.filter != "" {
		title += theme.InactiveStyle.Render(fmt.Sprintf(" [%s]", p.filter))
	}

	filtered := p.GetFiltered()

	if len(filtered) == 0 {
		content := theme.InactiveStyle.Render("No images")
		return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n\n" + content)
	}

	// Column widths
	tagW := p.width - 15
	if tagW < 15 {
		tagW = 15
	}
	sizeW := 10

	// Header
	header := fmt.Sprintf("%-*s %*s", tagW, "REPOSITORY:TAG", sizeW, "SIZE")
	headerStyled := theme.HighlightStyle.Render(header)

	// Rows
	visibleRows := p.height - 5
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Base text style for non-colored fields
	textStyle := lipgloss.NewStyle()

	var rows []string
	for i := p.offset; i < len(filtered) && i < p.offset+visibleRows; i++ {
		img := filtered[i]
		isSelected := i == p.selected

		tag := "<none>"
		if len(img.Tags) > 0 {
			tag = img.Tags[0]
		}
		tag = truncate(tag, tagW)

		size := docker.FormatBytesShort(uint64(img.Size))

		row := fmt.Sprintf("%-*s %*s", tagW, tag, sizeW, size)

		if isSelected {
			row = theme.SelectedStyle.Width(p.width - 4).Render(row)
		} else {
			row = textStyle.Width(p.width - 4).Render(row)
		}

		rows = append(rows, row)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{headerStyled, ""}, rows...)...)

	return style.Width(p.width - 2).Height(p.height - 2).Render(title + "\n" + content)
}
