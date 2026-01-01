package docker

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type Client struct {
	cli *client.Client
	mu  sync.RWMutex
}

type ContainerInfo struct {
	ID        string
	Name      string
	Image     string
	Status    string
	State     string
	Ports     string
	Created   time.Time
	CPUPerc   float64
	MemUsage  uint64
	MemLimit  uint64
	MemPerc   float64
	NetRx     uint64
	NetTx     uint64
	Autostart bool
}

type ImageInfo struct {
	ID      string
	Tags    []string
	Size    int64
	Created time.Time
}

type SystemStats struct {
	Containers        int
	ContainersRunning int
	ContainersPaused  int
	ContainersStopped int
	Images            int
	MemoryUsage       uint64
	MemoryLimit       uint64
	CPUUsage          float64
}

func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.cli.Ping(ctx)
	return err
}

func (c *Client) ListContainers(ctx context.Context) ([]ContainerInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var infos []ContainerInfo
	for _, cont := range containers {
		name := ""
		if len(cont.Names) > 0 {
			name = strings.TrimPrefix(cont.Names[0], "/")
		}

		ports := formatPorts(cont.Ports)

		infos = append(infos, ContainerInfo{
			ID:      cont.ID[:12],
			Name:    name,
			Image:   cont.Image,
			Status:  cont.Status,
			State:   cont.State,
			Ports:   ports,
			Created: time.Unix(cont.Created, 0),
		})
	}

	return infos, nil
}

func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*ContainerInfo, error) {
	stats, err := c.cli.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	var statsJSON container.StatsResponse
	if err := decodeStats(stats.Body, &statsJSON); err != nil {
		return nil, err
	}

	cpuPercent := calculateCPUPercent(&statsJSON)
	memUsage := statsJSON.MemoryStats.Usage
	memLimit := statsJSON.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100
	}

	var netRx, netTx uint64
	for _, net := range statsJSON.Networks {
		netRx += net.RxBytes
		netTx += net.TxBytes
	}

	return &ContainerInfo{
		ID:       containerID,
		CPUPerc:  cpuPercent,
		MemUsage: memUsage,
		MemLimit: memLimit,
		MemPerc:  memPercent,
		NetRx:    netRx,
		NetTx:    netTx,
	}, nil
}

func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	return c.cli.ContainerStart(ctx, containerID, container.StartOptions{})
}

func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RestartContainer(ctx context.Context, containerID string) error {
	timeout := 10
	return c.cli.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: &timeout})
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	return c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: force})
}

func (c *Client) GetContainerLogs(ctx context.Context, containerID string, lines int) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       string(rune(lines)),
		Timestamps: true,
	}

	// Fix: convert int to string properly
	options.Tail = formatInt(lines)

	reader, err := c.cli.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	// Strip docker log header bytes (8 bytes per line for multiplexed streams)
	return stripLogHeaders(string(logs)), nil
}

func (c *Client) StreamContainerLogs(ctx context.Context, containerID string, lines int) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       formatInt(lines),
		Follow:     true,
		Timestamps: true,
	}

	return c.cli.ContainerLogs(ctx, containerID, options)
}

func (c *Client) ListImages(ctx context.Context) ([]ImageInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	images, err := c.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}

	var infos []ImageInfo
	for _, img := range images {
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:19] // Get first 12 chars after sha256:
		}

		infos = append(infos, ImageInfo{
			ID:      id,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: time.Unix(img.Created, 0),
		})
	}

	return infos, nil
}

func (c *Client) PullImage(ctx context.Context, refStr string) (io.ReadCloser, error) {
	return c.cli.ImagePull(ctx, refStr, image.PullOptions{})
}

func (c *Client) RemoveImage(ctx context.Context, imageID string, force bool) error {
	_, err := c.cli.ImageRemove(ctx, imageID, image.RemoveOptions{Force: force})
	return err
}

func (c *Client) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	info, err := c.cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	return &SystemStats{
		Containers:        info.Containers,
		ContainersRunning: info.ContainersRunning,
		ContainersPaused:  info.ContainersPaused,
		ContainersStopped: info.ContainersStopped,
		Images:            info.Images,
		MemoryLimit:       uint64(info.MemTotal),
	}, nil
}

func (c *Client) SetRestartPolicy(ctx context.Context, containerID string, policy string) error {
	_, err := c.cli.ContainerUpdate(ctx, containerID, container.UpdateConfig{
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyMode(policy),
		},
	})
	return err
}

func (c *Client) GetContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return c.cli.ContainerInspect(ctx, containerID)
}
