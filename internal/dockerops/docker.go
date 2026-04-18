package dockerops

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/saeid/chunkdocker/internal/logging"
)

type Client struct {
	binary string
	logger *logging.Logger
}

type BuildOptions struct {
	ContextDir string
	Dockerfile string
	Image      string
	NoCache    bool
	Pull       bool
}

func New(binary string, logger *logging.Logger) (*Client, error) {
	resolved, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("docker binary %q not found: %w", binary, err)
	}
	return &Client{binary: resolved, logger: logger}, nil
}

func (c *Client) Build(ctx context.Context, opts BuildOptions) error {
	args := []string{"build", "--provenance=false", "--sbom=false", "-f", opts.Dockerfile, "-t", opts.Image}
	if opts.NoCache {
		args = append(args, "--no-cache")
	}
	if opts.Pull {
		args = append(args, "--pull")
	}
	args = append(args, opts.ContextDir)

	c.logger.Infof("building image %s", opts.Image)
	if _, err := c.run(ctx, args...); err != nil {
		return fmt.Errorf("failed to build image %s: %w", opts.Image, err)
	}
	return nil
}

func (c *Client) Push(ctx context.Context, image string) error {
	c.logger.Infof("pushing image %s", image)
	if _, err := c.run(ctx, "push", image); err != nil {
		return fmt.Errorf("failed to push image %s: %w", image, err)
	}
	return nil
}

func (c *Client) Pull(ctx context.Context, image string) error {
	c.logger.Infof("pulling image %s", image)
	if _, err := c.run(ctx, "pull", image); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}
	return nil
}

func (c *Client) CreateContainer(ctx context.Context, image string) (string, error) {
	output, err := c.run(ctx, "create", image, "/__chunkdocker_noop__")
	if err != nil {
		return "", fmt.Errorf("failed to create container from image %s: %w", image, err)
	}
	return strings.TrimSpace(output), nil
}

func (c *Client) CopyFromContainer(ctx context.Context, containerID, sourcePath, destinationPath string) error {
	if _, err := c.run(ctx, "cp", fmt.Sprintf("%s:%s", containerID, sourcePath), destinationPath); err != nil {
		return fmt.Errorf("failed to copy %s from container %s: %w", sourcePath, containerID, err)
	}
	return nil
}

func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	if containerID == "" {
		return nil
	}
	if _, err := c.run(ctx, "rm", "-f", containerID); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, c.binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w: %s", c.binary, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
