package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/seb07-cloud/dktop/internal/config"
	"github.com/seb07-cloud/dktop/internal/daemon"
	"github.com/seb07-cloud/dktop/internal/docker"
	"github.com/seb07-cloud/dktop/internal/ui"
	"github.com/seb07-cloud/dktop/internal/version"
)

const logo = `
    ██████╗ ██╗  ██╗████████╗ ██████╗ ██████╗
    ██╔══██╗██║ ██╔╝╚══██╔══╝██╔═══██╗██╔══██╗
    ██║  ██║█████╔╝    ██║   ██║   ██║██████╔╝
    ██║  ██║██╔═██╗    ██║   ██║   ██║██╔═══╝
    ██████╔╝██║  ██╗   ██║   ╚██████╔╝██║
    ╚═════╝ ╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝
`

const usage = `dktop - Docker container manager with btop-style interface

Usage:
  dktop              Start the interactive TUI
  dktop daemon       Run as daemon (monitors autostart containers)
  dktop version      Show version information
  dktop help         Show this help message

Keybindings:
  Tab        Switch between panels
  j/k, ↑/↓   Navigate lists
  s          Start container
  x          Stop container
  r          Restart container
  d          Delete container/image
  a          Toggle autostart
  p          Pull image (in images panel)
  Enter      View full logs
  /          Filter
  G          Scroll to bottom (in logs)
  q          Quit
`

func getConfigPathHelp() string {
	path, err := config.GetConfigPath()
	if err != nil {
		return "Config: <unable to determine path>"
	}
	return "Config: " + path
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "daemon":
			runDaemon()
			return
		case "version":
			fmt.Printf("dktop version %s\n", version.String())
			return
		case "help", "-h", "--help":
			fmt.Print(logo)
			fmt.Print(usage)
			fmt.Println(getConfigPathHelp())
			return
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			fmt.Print(usage)
			os.Exit(1)
		}
	}

	runTUI()
}

func runTUI() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = &config.DefaultConfig
	}

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure Docker is running and accessible.")
		os.Exit(1)
	}
	defer dockerClient.Close()

	// Test connection
	ctx := context.Background()
	if err := dockerClient.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		fmt.Fprintln(os.Stderr, "Make sure Docker is running and accessible.")
		os.Exit(1)
	}

	// Create and run the app
	app := ui.NewApp(dockerClient, cfg)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running dktop: %v\n", err)
		os.Exit(1)
	}
}

func runDaemon() {
	fmt.Print(logo)
	fmt.Println("Starting dktop daemon...")

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = &config.DefaultConfig
	}

	// Create Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		os.Exit(1)
	}
	defer dockerClient.Close()

	// Test connection
	ctx := context.Background()
	if err := dockerClient.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker: %v\n", err)
		os.Exit(1)
	}

	// Create and run daemon
	d := daemon.New(dockerClient, cfg)
	if err := d.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Daemon error: %v\n", err)
		os.Exit(1)
	}
}
