package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/seb07-cloud/dktop/internal/config"
	"github.com/seb07-cloud/dktop/internal/docker"
)

type Daemon struct {
	client   *docker.Client
	config   *config.Config
	interval time.Duration
	logger   *log.Logger
}

func New(client *docker.Client, cfg *config.Config) *Daemon {
	return &Daemon{
		client:   client,
		config:   cfg,
		interval: 30 * time.Second,
		logger:   log.New(os.Stdout, "[dktop-daemon] ", log.LstdFlags),
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	d.logger.Println("Starting dktop daemon...")
	d.logger.Printf("Monitoring %d containers for autostart", len(d.config.AutostartList))

	// Set up signal handling (os.Interrupt works on both Windows and Unix)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	// Initial check
	d.checkAndStartContainers(ctx)

	for {
		select {
		case <-ctx.Done():
			d.logger.Println("Daemon stopping (context cancelled)")
			return ctx.Err()
		case <-sigChan:
			d.logger.Println("Daemon stopping (signal received)")
			return nil
		case <-ticker.C:
			d.checkAndStartContainers(ctx)
		}
	}
}

func (d *Daemon) checkAndStartContainers(ctx context.Context) {
	// Reload config to pick up changes
	newConfig, err := config.Load()
	if err == nil {
		d.config = newConfig
	}

	if len(d.config.AutostartList) == 0 {
		return
	}

	containers, err := d.client.ListContainers(ctx)
	if err != nil {
		d.logger.Printf("Error listing containers: %v", err)
		return
	}

	// Build a map of container name/ID to container info
	containerMap := make(map[string]docker.ContainerInfo)
	for _, c := range containers {
		containerMap[c.ID] = c
		containerMap[c.Name] = c
	}

	// Check each autostart container
	for _, name := range d.config.AutostartList {
		container, exists := containerMap[name]
		if !exists {
			d.logger.Printf("Autostart container not found: %s", name)
			continue
		}

		if container.State != "running" {
			d.logger.Printf("Starting container: %s (was %s)", name, container.State)
			if err := d.client.StartContainer(ctx, container.ID); err != nil {
				d.logger.Printf("Error starting container %s: %v", name, err)
			} else {
				d.logger.Printf("Successfully started container: %s", name)
			}
		}
	}
}

// RunOnce performs a single check and start operation
func (d *Daemon) RunOnce(ctx context.Context) error {
	d.checkAndStartContainers(ctx)
	return nil
}

// Status returns the current daemon status
func (d *Daemon) Status() string {
	return fmt.Sprintf("Monitoring %d containers, check interval: %v",
		len(d.config.AutostartList), d.interval)
}
