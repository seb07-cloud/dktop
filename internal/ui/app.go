package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/seb/dktop/internal/config"
	"github.com/seb/dktop/internal/docker"
	"github.com/seb/dktop/internal/theme"
	"github.com/seb/dktop/internal/version"
)

type Panel int

const (
	PanelStats Panel = iota
	PanelImages
	PanelContainers
	PanelLogs
)

// Logo banner for the top of the app
const logoBanner = `    ██████╗ ██╗  ██╗████████╗ ██████╗ ██████╗
    ██╔══██╗██║ ██╔╝╚══██╔══╝██╔═══██╗██╔══██╗
    ██║  ██║█████╔╝    ██║   ██║   ██║██████╔╝
    ██║  ██║██╔═██╗    ██║   ██║   ██║██╔═══╝
    ██████╔╝██║  ██╗   ██║   ╚██████╔╝██║
    ╚═════╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝`

type Mode int

const (
	ModeNormal Mode = iota
	ModeFilter
	ModePullImage
)

type App struct {
	// Dimensions
	width  int
	height int

	// Panels
	statsPanel      *StatsPanel
	imagesPanel     *ImagesPanel
	containersPanel *ContainersPanel
	logsPanel       *LogsPanel
	helpBar         *HelpBar

	// State
	activePanel Panel
	mode        Mode
	filterInput textinput.Model
	pullInput   textinput.Model
	err         error

	// Docker client
	dockerClient *docker.Client
	config       *config.Config

	// Data
	containers  []docker.ContainerInfo
	images      []docker.ImageInfo
	systemStats *docker.SystemStats

	// Refresh
	refreshInterval time.Duration

	// Cached renders
	renderedLogo string
}

// Messages
type tickMsg time.Time
type containersMsg []docker.ContainerInfo
type imagesMsg []docker.ImageInfo
type systemStatsMsg *docker.SystemStats
type containerStatsMsg struct {
	id    string
	stats *docker.ContainerInfo
}
type logsMsg string
type errMsg error

func NewApp(dockerClient *docker.Client, cfg *config.Config) *App {
	filterInput := textinput.New()
	filterInput.Placeholder = "Filter..."
	filterInput.CharLimit = 50

	pullInput := textinput.New()
	pullInput.Placeholder = "image:tag"
	pullInput.CharLimit = 100

	return &App{
		statsPanel:      NewStatsPanel(),
		imagesPanel:     NewImagesPanel(),
		containersPanel: NewContainersPanel(),
		logsPanel:       NewLogsPanel(),
		helpBar:         NewHelpBar(),
		activePanel:     PanelContainers,
		mode:            ModeNormal,
		filterInput:     filterInput,
		pullInput:       pullInput,
		dockerClient:    dockerClient,
		config:          cfg,
		refreshInterval: time.Duration(cfg.RefreshRate) * time.Millisecond,
		renderedLogo:    "", // Will be set on first WindowSizeMsg
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.tickCmd(),
		a.fetchContainers(),
		a.fetchImages(),
		a.fetchSystemStats(),
	)
}

func (a *App) updateLogo() {
	logo := theme.LogoStyle.Render(logoBanner)
	versionStr := theme.InactiveStyle.Render("v" + version.String())

	// Calculate padding to right-align version
	logoWidth := lipgloss.Width(logo)
	versionWidth := lipgloss.Width(versionStr)
	padding := a.width - logoWidth - versionWidth - 2
	if padding < 1 {
		padding = 1
	}

	// Place version on the first line, right-aligned
	spacer := lipgloss.NewStyle().Width(padding).Render("")
	a.renderedLogo = lipgloss.JoinHorizontal(lipgloss.Top, logo, spacer, versionStr)
}

func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(a.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (a *App) fetchContainers() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		containers, err := a.dockerClient.ListContainers(ctx)
		if err != nil {
			return errMsg(err)
		}

		// Mark autostart containers
		for i := range containers {
			containers[i].Autostart = a.config.IsAutostart(containers[i].ID) ||
				a.config.IsAutostart(containers[i].Name)
		}

		return containersMsg(containers)
	}
}

func (a *App) fetchContainerStats(containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stats, err := a.dockerClient.GetContainerStats(ctx, containerID)
		if err != nil {
			return nil // Silently ignore stats errors
		}

		return containerStatsMsg{id: containerID, stats: stats}
	}
}

func (a *App) fetchImages() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		images, err := a.dockerClient.ListImages(ctx)
		if err != nil {
			return errMsg(err)
		}
		return imagesMsg(images)
	}
}

func (a *App) fetchSystemStats() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stats, err := a.dockerClient.GetSystemStats(ctx)
		if err != nil {
			return errMsg(err)
		}
		return systemStatsMsg(stats)
	}
}

func (a *App) fetchLogs() tea.Cmd {
	selected := a.containersPanel.GetSelected()
	if selected == nil {
		return nil
	}

	containerID := selected.ID
	containerName := selected.Name
	a.logsPanel.SetContainerName(containerName)

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		logs, err := a.dockerClient.GetContainerLogs(ctx, containerID, 100)
		if err != nil {
			return errMsg(err)
		}
		return logsMsg(logs)
	}
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLogo()
		a.updatePanelSizes()

	case tea.KeyMsg:
		cmd := a.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tickMsg:
		cmds = append(cmds, a.tickCmd())
		cmds = append(cmds, a.fetchContainers())
		cmds = append(cmds, a.fetchSystemStats())

		// Fetch stats for running containers
		for _, c := range a.containers {
			if c.State == "running" {
				cmds = append(cmds, a.fetchContainerStats(c.ID))
			}
		}

		// Fetch logs for selected container
		if a.activePanel == PanelLogs || a.activePanel == PanelContainers {
			cmds = append(cmds, a.fetchLogs())
		}

	case containersMsg:
		a.containers = msg
		a.containersPanel.Update(a.containers)
		a.updatePanelSizes() // Resize panels based on container count
		a.err = nil          // Clear any stale errors (e.g., from deleted containers)

	case containerStatsMsg:
		// Update container stats
		for i := range a.containers {
			if a.containers[i].ID == msg.id {
				a.containers[i].CPUPerc = msg.stats.CPUPerc
				a.containers[i].MemUsage = msg.stats.MemUsage
				a.containers[i].MemLimit = msg.stats.MemLimit
				a.containers[i].MemPerc = msg.stats.MemPerc
				a.containers[i].NetRx = msg.stats.NetRx
				a.containers[i].NetTx = msg.stats.NetTx
				break
			}
		}
		a.containersPanel.Update(a.containers)

	case imagesMsg:
		a.images = msg
		a.imagesPanel.Update(a.images)

	case systemStatsMsg:
		a.systemStats = msg
		// Calculate total CPU/Memory from running containers
		if a.systemStats != nil {
			var totalCPU float64
			var totalMem uint64
			for _, c := range a.containers {
				if c.State == "running" {
					totalCPU += c.CPUPerc
					totalMem += c.MemUsage
				}
			}
			a.systemStats.CPUUsage = totalCPU
			a.systemStats.MemoryUsage = totalMem
		}
		a.statsPanel.Update(a.systemStats)

	case logsMsg:
		a.logsPanel.Update(string(msg))

	case errMsg:
		a.err = msg
	}

	// Update text inputs in filter/pull mode
	if a.mode == ModeFilter {
		var cmd tea.Cmd
		a.filterInput, cmd = a.filterInput.Update(msg)
		cmds = append(cmds, cmd)
	} else if a.mode == ModePullImage {
		var cmd tea.Cmd
		a.pullInput, cmd = a.pullInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

func (a *App) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// Handle filter mode
	if a.mode == ModeFilter {
		switch msg.String() {
		case "enter":
			filter := a.filterInput.Value()
			if a.activePanel == PanelContainers {
				a.containersPanel.SetFilter(filter)
			} else if a.activePanel == PanelImages {
				a.imagesPanel.SetFilter(filter)
			}
			a.mode = ModeNormal
			a.filterInput.Blur()
		case "esc":
			a.mode = ModeNormal
			a.filterInput.Blur()
			a.filterInput.SetValue("")
		}
		return nil
	}

	// Handle pull image mode
	if a.mode == ModePullImage {
		switch msg.String() {
		case "enter":
			imageName := a.pullInput.Value()
			a.mode = ModeNormal
			a.pullInput.Blur()
			a.pullInput.SetValue("")
			return a.pullImage(imageName)
		case "esc":
			a.mode = ModeNormal
			a.pullInput.Blur()
			a.pullInput.SetValue("")
		}
		return nil
	}

	// Normal mode
	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit

	case "tab":
		a.cyclePanel()

	case "j", "down":
		a.handleDown()

	case "k", "up":
		a.handleUp()

	case "s":
		if a.activePanel == PanelContainers {
			return a.startSelectedContainer()
		}

	case "x":
		if a.activePanel == PanelContainers {
			return a.stopSelectedContainer()
		}

	case "r":
		if a.activePanel == PanelContainers {
			return a.restartSelectedContainer()
		}

	case "d":
		if a.activePanel == PanelContainers {
			return a.deleteSelectedContainer()
		} else if a.activePanel == PanelImages {
			return a.deleteSelectedImage()
		}

	case "a":
		if a.activePanel == PanelContainers {
			return a.toggleAutostart()
		}

	case "p":
		if a.activePanel == PanelImages {
			a.mode = ModePullImage
			a.pullInput.Focus()
			return textinput.Blink
		}

	case "enter":
		if a.activePanel == PanelContainers {
			a.activePanel = PanelLogs
			a.updatePanelActive()
			return a.fetchLogs()
		}

	case "esc":
		if a.activePanel == PanelLogs {
			a.activePanel = PanelContainers
			a.updatePanelActive()
		}

	case "/":
		a.mode = ModeFilter
		a.filterInput.Focus()
		return textinput.Blink

	case "G":
		if a.activePanel == PanelLogs {
			a.logsPanel.ScrollToBottom()
		}
	}

	return nil
}

func (a *App) cyclePanel() {
	panels := []Panel{PanelContainers, PanelImages, PanelLogs}
	for i, p := range panels {
		if p == a.activePanel {
			a.activePanel = panels[(i+1)%len(panels)]
			break
		}
	}
	a.updatePanelActive()
}

func (a *App) handleDown() {
	switch a.activePanel {
	case PanelContainers:
		a.containersPanel.MoveDown()
	case PanelImages:
		a.imagesPanel.MoveDown()
	case PanelLogs:
		a.logsPanel.ScrollDown()
	}
}

func (a *App) handleUp() {
	switch a.activePanel {
	case PanelContainers:
		a.containersPanel.MoveUp()
	case PanelImages:
		a.imagesPanel.MoveUp()
	case PanelLogs:
		a.logsPanel.ScrollUp()
	}
}

func (a *App) updatePanelActive() {
	a.statsPanel.SetActive(a.activePanel == PanelStats)
	a.imagesPanel.SetActive(a.activePanel == PanelImages)
	a.containersPanel.SetActive(a.activePanel == PanelContainers)
	a.logsPanel.SetActive(a.activePanel == PanelLogs)
}

func (a *App) updatePanelSizes() {
	// Layout:
	// ┌─────────────────────────────────────────────────────────────────┐
	// │  Logo Banner (6 lines)                                          │
	// ├─────────────────────────────────────┬──────────────────────────┤
	// │  Docker System Stats (CPU/Mem/Net)  │     Images List          │
	// ├─────────────────────────────────────┴──────────────────────────┤
	// │                    Container List                              │
	// ├────────────────────────────────────────────────────────────────┤
	// │                    Logs (selected container)                   │
	// └────────────────────────────────────────────────────────────────┘

	bannerHeight := 6 // Logo banner takes 6 lines
	topHeight := 17   // Stats/Images panels height (content 15 + border 2)
	helpHeight := 1   // Help bar at bottom

	// Total available for containers + logs
	remainingHeight := a.height - bannerHeight - topHeight - helpHeight
	if remainingHeight < 2 {
		remainingHeight = 2
	}

	// Container panel height based on number of containers
	// Add 5 for: border (2) + title (1) + header (1) + empty line after header (1)
	containerCount := len(a.containers)
	if containerCount < 1 {
		containerCount = 1 // Minimum 1 row for "No containers"
	}
	containerHeight := containerCount + 5

	// Logs get minimum 5 lines, containers get the rest
	minLogsHeight := 5
	if minLogsHeight > remainingHeight-1 {
		minLogsHeight = remainingHeight - 1
	}

	// Cap container height to leave room for logs
	maxContainerHeight := remainingHeight - minLogsHeight
	if containerHeight > maxContainerHeight {
		containerHeight = maxContainerHeight
	}
	if containerHeight < 1 {
		containerHeight = 1
	}

	// Logs get exactly the remaining space
	logsHeight := remainingHeight - containerHeight
	if logsHeight < 1 {
		logsHeight = 1
	}

	statsWidth := a.width / 2
	imagesWidth := a.width - statsWidth

	a.statsPanel.SetSize(statsWidth, topHeight)
	a.imagesPanel.SetSize(imagesWidth, topHeight)
	a.containersPanel.SetSize(a.width, containerHeight)
	a.logsPanel.SetSize(a.width, logsHeight)
	a.helpBar.SetWidth(a.width)

	a.updatePanelActive()
}

func (a *App) startSelectedContainer() tea.Cmd {
	selected := a.containersPanel.GetSelected()
	if selected == nil || selected.State == "running" {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.dockerClient.StartContainer(ctx, selected.ID); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (a *App) stopSelectedContainer() tea.Cmd {
	selected := a.containersPanel.GetSelected()
	if selected == nil || selected.State != "running" {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.dockerClient.StopContainer(ctx, selected.ID); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (a *App) restartSelectedContainer() tea.Cmd {
	selected := a.containersPanel.GetSelected()
	if selected == nil {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.dockerClient.RestartContainer(ctx, selected.ID); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (a *App) deleteSelectedContainer() tea.Cmd {
	selected := a.containersPanel.GetSelected()
	if selected == nil {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Force remove if running
		force := selected.State == "running"
		if err := a.dockerClient.RemoveContainer(ctx, selected.ID, force); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (a *App) deleteSelectedImage() tea.Cmd {
	selected := a.imagesPanel.GetSelected()
	if selected == nil {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := a.dockerClient.RemoveImage(ctx, selected.ID, false); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (a *App) toggleAutostart() tea.Cmd {
	selected := a.containersPanel.GetSelected()
	if selected == nil {
		return nil
	}

	// Toggle in config
	if a.config.IsAutostart(selected.Name) {
		a.config.RemoveAutostart(selected.Name)
	} else {
		a.config.AddAutostart(selected.Name)
	}

	// Save config
	_ = a.config.Save()

	// Update restart policy
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		policy := "no"
		if a.config.IsAutostart(selected.Name) {
			policy = "always"
		}

		if err := a.dockerClient.SetRestartPolicy(ctx, selected.ID, policy); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (a *App) pullImage(imageName string) tea.Cmd {
	if imageName == "" {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		reader, err := a.dockerClient.PullImage(ctx, imageName)
		if err != nil {
			return errMsg(err)
		}
		defer reader.Close()

		// Read to completion (we could show progress but keeping it simple)
		buf := make([]byte, 1024)
		for {
			_, err := reader.Read(buf)
			if err != nil {
				break
			}
		}

		return nil
	}
}

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	// Top row: Stats | Images
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, a.statsPanel.View(), a.imagesPanel.View())

	// Middle: Containers
	containersView := a.containersPanel.View()

	// Bottom: Logs
	logsView := a.logsPanel.View()

	// Help bar
	helpView := a.helpBar.View(a.activePanel)

	// Input bar (if in filter/pull mode)
	inputBar := ""
	if a.mode == ModeFilter {
		inputBar = theme.HighlightStyle.Render("Filter: ") + a.filterInput.View()
	} else if a.mode == ModePullImage {
		inputBar = theme.HighlightStyle.Render("Pull image: ") + a.pullInput.View()
	}

	// Error display
	errBar := ""
	if a.err != nil {
		errBar = theme.HighUsageStyle.Render(fmt.Sprintf("Error: %v", a.err))
	}

	// Combine all using cached logo
	var view string
	if inputBar != "" {
		view = lipgloss.JoinVertical(lipgloss.Left, a.renderedLogo, topRow, containersView, logsView, inputBar, helpView)
	} else if errBar != "" {
		view = lipgloss.JoinVertical(lipgloss.Left, a.renderedLogo, topRow, containersView, logsView, errBar, helpView)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left, a.renderedLogo, topRow, containersView, logsView, helpView)
	}

	return view
}
