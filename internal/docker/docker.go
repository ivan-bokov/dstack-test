package docker

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockerCLI "github.com/docker/docker/client"
	"github.com/ivan-bokov/dstack-test/internal/stacktrace"
)

type Container struct {
	Image  string
	Shell  string
	client *dockerCLI.Client
	logs   chan string
	id     string
	stop   bool
}

func New(image string, shell string) (*Container, error) {
	cli, err := dockerCLI.NewClientWithOpts(dockerCLI.FromEnv)
	if err != nil {
		return nil, stacktrace.Wrap(err)
	}
	return &Container{
		Image:  image,
		Shell:  shell,
		client: cli,
		logs:   make(chan string, 100),
	}, nil
}

func (c *Container) pull(ctx context.Context) error {
	_, err := c.client.ImagePull(ctx, c.Image, types.ImagePullOptions{})
	if err != nil {
		return stacktrace.Wrap(err)
	}
	return nil
}

func (c *Container) Logs() chan string {
	return c.logs
}

func (c *Container) Run(ctx context.Context) error {
	err := c.pull(ctx)
	if err != nil {
		return stacktrace.Wrap(err)
	}
	resp, err := c.client.ContainerCreate(ctx, &container.Config{
		Tty:         true,
		Cmd:         []string{"sh", "-c", c.Shell},
		Image:       c.Image,
		ArgsEscaped: true,
	}, nil, nil, nil, "")
	if err != nil {
		return stacktrace.Wrap(err)
	}
	c.id = resp.ID
	if err = c.client.ContainerStart(ctx, c.id, types.ContainerStartOptions{}); err != nil {
		return stacktrace.Wrap(err)
	}
	go c.logging(ctx)
	return nil
}

func (c *Container) Wait(ctx context.Context) {
	waitC, errC := c.client.ContainerWait(ctx, c.id, "")
	select {
	case <-waitC:
		return
	case err := <-errC:
		if !errors.Is(err, context.Canceled) {
			log.Println(fmt.Sprintf("[ERROR] Error in container: %s error: %#v", c.id, err))
		}
		return
	}
}

func (c *Container) logging(ctx context.Context) {
	out, err := c.client.ContainerLogs(ctx, c.id, types.ContainerLogsOptions{
		ShowStdout: true,
		Follow:     true,
	})
	if err != nil {
		log.Println(fmt.Sprintf("[ERROR] unable to fetch the container logs: %s error: %#v", c.id, err))
		return
	}
	defer out.Close()
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			c.logs <- scanner.Text()
		}
	}
}

func (c *Container) Close() {
	var err error

	close(c.logs)
	if !c.stop {
		d := 1 * time.Second
		if err = c.client.ContainerStop(context.Background(), c.id, &d); err != nil {
			log.Println(fmt.Sprintf("[ERROR] unable to stop container: %s error: %#v", c.id, err))
		}
	}
	if err = c.client.ContainerRemove(context.Background(), c.id, types.ContainerRemoveOptions{Force: true}); err != nil {
		log.Println(fmt.Sprintf("[ERROR] unable to remove container: %s error: %#v", c.id, err))
	}
	if err = c.client.Close(); err != nil {
		log.Println(fmt.Sprintf("[ERROR] unable to close Docker client: %s error: %#v", c.id, err))
	}
}
