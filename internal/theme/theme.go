package theme

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colorful terminal color scheme using bright ANSI 256 colors
var (
	// Bright, vibrant colors
	Green  = lipgloss.Color("46")  // Bright green
	Yellow = lipgloss.Color("226") // Bright yellow
	Red    = lipgloss.Color("196") // Bright red
	Blue   = lipgloss.Color("39")  // Bright blue
	Purple = lipgloss.Color("171") // Bright magenta
	Cyan   = lipgloss.Color("51")  // Bright cyan
	Orange = lipgloss.Color("208") // Orange
	Pink   = lipgloss.Color("205") // Pink
	White  = lipgloss.Color("255") // Bright white
	Gray   = lipgloss.Color("244") // Light gray

	// Styles - colorful and vibrant
	BaseStyle = lipgloss.NewStyle()

	TitleStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	BorderStyle = lipgloss.NewStyle().
			Foreground(Purple)

	HighlightStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	SelectedStyle = lipgloss.NewStyle().
			Background(Blue).
			Foreground(White).
			Bold(true)

	InactiveStyle = lipgloss.NewStyle().
			Foreground(Gray)

	// Status styles
	RunningStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	StoppedStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	PausedStyle = lipgloss.NewStyle().
			Foreground(Orange).
			Bold(true)

	// Resource usage styles (for gradient display)
	LowUsageStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	MediumUsageStyle = lipgloss.NewStyle().
				Foreground(Yellow).
				Bold(true)

	HighUsageStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	// Panel styles
	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(0, 1)

	ActivePanelStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Cyan).
				Padding(0, 1)

	// Help bar style
	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Pink).
			Bold(true)

	// Logo style - bright cyan with bold
	LogoStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)
)

// GetUsageStyle returns the appropriate style based on usage percentage
func GetUsageStyle(percent float64) lipgloss.Style {
	switch {
	case percent >= 80:
		return HighUsageStyle
	case percent >= 50:
		return MediumUsageStyle
	default:
		return LowUsageStyle
	}
}

// GetUsageColor returns the appropriate color based on usage percentage
func GetUsageColor(percent float64) lipgloss.Color {
	switch {
	case percent >= 80:
		return Red
	case percent >= 50:
		return Yellow
	default:
		return Green
	}
}

// RenderSparkline creates a multi-row line graph from historical data
// Uses box-drawing characters to create a smooth continuous line
// Auto-scales based on min/max values in the data
func RenderSparkline(data []float64, width int, color lipgloss.Color) string {
	style := lipgloss.NewStyle().Foreground(color)
	height := 3 // 3 rows for the graph

	if width < 1 {
		width = 1
	}

	// Create empty grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	if len(data) == 0 {
		var b strings.Builder
		for i := 0; i < height; i++ {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(style.Render(strings.Repeat("─", width)))
		}
		return b.String()
	}

	// Resample data to fit width
	var sampledData []float64
	if len(data) >= width {
		sampledData = make([]float64, width)
		for i := 0; i < width; i++ {
			idx := i * len(data) / width
			sampledData[i] = data[idx]
		}
	} else {
		sampledData = make([]float64, width)
		offset := width - len(data)
		for i := 0; i < offset; i++ {
			sampledData[i] = -1
		}
		for i := 0; i < len(data); i++ {
			sampledData[offset+i] = data[i]
		}
	}

	// Find min and max for auto-scaling
	minVal := float64(100)
	maxVal := float64(0)
	hasData := false
	for _, val := range sampledData {
		if val < 0 {
			continue
		}
		hasData = true
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}

	if !hasData {
		var b strings.Builder
		for i := 0; i < height; i++ {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(style.Render(strings.Repeat(" ", width)))
		}
		return b.String()
	}

	rangeVal := maxVal - minVal
	if rangeVal < 1 {
		rangeVal = 1
	}

	// Convert to row positions (0 = bottom, height-1 = top)
	rows := make([]int, width)
	for i, val := range sampledData {
		if val < 0 {
			rows[i] = -1
			continue
		}
		normalized := (val - minVal) / rangeVal
		if normalized < 0 {
			normalized = 0
		}
		if normalized > 1 {
			normalized = 1
		}
		rows[i] = int(normalized * float64(height-1))
	}

	// Draw the line using box-drawing characters
	// Characters: ─ │ ╭ ╮ ╯ ╰
	for x := 0; x < width; x++ {
		if rows[x] < 0 {
			continue
		}

		currRow := rows[x]
		prevRow := currRow
		nextRow := currRow

		if x > 0 && rows[x-1] >= 0 {
			prevRow = rows[x-1]
		}
		if x < width-1 && rows[x+1] >= 0 {
			nextRow = rows[x+1]
		}

		// Invert row for grid (0 = top in grid)
		y := height - 1 - currRow

		// Determine the character based on prev/curr/next positions
		if prevRow == currRow && nextRow == currRow {
			// Flat line
			grid[y][x] = '─'
		} else if prevRow < currRow && nextRow < currRow {
			// Peak (coming up, going down)
			grid[y][x] = '╮'
			// Draw vertical line down to next
			for dy := y + 1; dy < height && dy <= height-1-nextRow; dy++ {
				if grid[dy][x] == ' ' {
					grid[dy][x] = '│'
				}
			}
		} else if prevRow > currRow && nextRow > currRow {
			// Valley (coming down, going up)
			grid[y][x] = '╰'
		} else if prevRow < currRow && nextRow == currRow {
			// Coming up, then flat
			grid[y][x] = '╭'
		} else if prevRow == currRow && nextRow < currRow {
			// Flat, then going down
			grid[y][x] = '╮'
			// Draw vertical line down
			for dy := y + 1; dy <= height-1-nextRow; dy++ {
				if grid[dy][x] == ' ' {
					grid[dy][x] = '│'
				}
			}
		} else if prevRow > currRow && nextRow == currRow {
			// Coming down, then flat
			grid[y][x] = '╯'
		} else if prevRow == currRow && nextRow > currRow {
			// Flat, then going up
			grid[y][x] = '╰'
			// Draw vertical line up to next position
			for dy := y - 1; dy >= height-1-nextRow; dy-- {
				if grid[dy][x] == ' ' {
					grid[dy][x] = '│'
				}
			}
		} else if prevRow < currRow && nextRow > currRow {
			// Coming up, continuing up
			grid[y][x] = '│'
		} else if prevRow > currRow && nextRow < currRow {
			// Coming down, continuing down
			grid[y][x] = '│'
		} else {
			grid[y][x] = '─'
		}
	}

	// Render the grid
	var b strings.Builder
	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(style.Render(string(grid[i])))
	}

	return b.String()
}

// RenderLineGraph creates a continuous line graph from historical data
// Uses braille-style block characters: ▁▂▃▄▅▆▇█
func RenderLineGraph(data []float64, width, height int, color lipgloss.Color) string {
	if height < 1 {
		height = 1
	}
	if width < 1 {
		width = 1
	}

	// Block characters for different heights (8 levels)
	blocks := []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Create empty grid
	grid := make([][]rune, height)
	for i := range grid {
		grid[i] = make([]rune, width)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	if len(data) == 0 {
		// Return empty graph
		var b strings.Builder
		style := lipgloss.NewStyle().Foreground(color)
		for i := 0; i < height; i++ {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(style.Render(strings.Repeat(" ", width)))
		}
		return b.String()
	}

	// Resample data to fit width
	sampledData := make([]float64, width)
	if len(data) >= width {
		// More data than width, sample evenly
		for i := 0; i < width; i++ {
			idx := i * len(data) / width
			sampledData[i] = data[idx]
		}
	} else {
		// Less data than width, right-align the data
		offset := width - len(data)
		for i := 0; i < offset; i++ {
			sampledData[i] = 0
		}
		for i := 0; i < len(data); i++ {
			sampledData[offset+i] = data[i]
		}
	}

	// Draw the graph - each column gets filled from bottom up
	for x, val := range sampledData {
		if val <= 0 {
			continue
		}

		// Calculate how many full rows and the partial block
		// val is 0-100 percentage
		normalizedVal := val / 100.0
		if normalizedVal > 1.0 {
			normalizedVal = 1.0
		}

		totalHeight := normalizedVal * float64(height)
		fullRows := int(totalHeight)
		partialAmount := totalHeight - float64(fullRows)

		// Fill full blocks from bottom
		for y := height - 1; y > height-1-fullRows && y >= 0; y-- {
			grid[y][x] = blocks[8] // Full block
		}

		// Add partial block on top
		if fullRows < height && partialAmount > 0 {
			blockIndex := int(partialAmount * 8)
			if blockIndex > 0 {
				y := height - 1 - fullRows
				if y >= 0 {
					grid[y][x] = blocks[blockIndex]
				}
			}
		}
	}

	// Render the grid with color
	var b strings.Builder
	style := lipgloss.NewStyle().Foreground(color)

	for i := 0; i < height; i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(style.Render(string(grid[i])))
	}

	return b.String()
}

// RenderProgressBar creates a colored progress bar
func RenderProgressBar(percent float64, width int) string {
	filled := int(float64(width) * percent / 100)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	color := GetUsageColor(percent)
	filledStyle := lipgloss.NewStyle().Foreground(color)
	emptyStyle := lipgloss.NewStyle().Foreground(Gray)

	// Use strings.Builder for better performance
	var b strings.Builder
	b.Grow(width * 4) // Estimate 4 bytes per character for unicode

	filledChar := filledStyle.Render("█")
	emptyChar := emptyStyle.Render("░")

	for i := 0; i < filled; i++ {
		b.WriteString(filledChar)
	}
	for i := filled; i < width; i++ {
		b.WriteString(emptyChar)
	}

	return b.String()
}
