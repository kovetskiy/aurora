package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/karma-go"
)

const (
	ImageLabelKey = "io.reconquest/aurora"
)

type Cloud struct {
	client    *client.Client
	BaseImage string

	mutex     sync.Mutex
	resources ConfigResources
	threads   int
	cpuNext   int
}

func NewCloud(baseImage string, resources ConfigResources, threads int) (*Cloud, error) {
	var err error

	cloud := &Cloud{}
	cloud.client, err = client.NewEnvClient()
	cloud.BaseImage = baseImage
	cloud.resources = resources

	if threads == 0 {
		threads = runtime.NumCPU()
	}

	cloud.threads = threads

	return cloud, err
}

func (cloud *Cloud) getNextCPU() string {
	if cloud.resources.CPU == 0 {
		return ""
	}

	cloud.mutex.Lock()
	defer cloud.mutex.Unlock()

	start := cloud.cpuNext
	end := start + cloud.resources.CPU - 1
	cloud.cpuNext = (cloud.cpuNext + cloud.resources.CPU) % cloud.threads

	if start == end {
		return strconv.Itoa(start)
	} else {
		return fmt.Sprintf("%d-%d", start, end)
	}
}

func (cloud *Cloud) CreateContainer(
	bufferDir string,
	containerName string,
	packageName string,
	cloneURL string,
	subdir string,
) (string, error) {
	config := &container.Config{
		Image: cloud.BaseImage,
		Labels: map[string]string{
			ImageLabelKey: version,
		},
		Tty: true,
		Env: []string{
			fmt.Sprintf("AURORA_PACKAGE=%s", packageName),
			fmt.Sprintf("AURORA_CLONE_URL=%s", cloneURL),
			fmt.Sprintf("AURORA_SUBDIR=%s", subdir),
		},
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/buffer", bufferDir),
		},
	}

	if cloud.resources.CPU > 0 {
		hostConfig.Resources.CpusetCpus = cloud.getNextCPU()
	}

	created, err := cloud.client.ContainerCreate(
		context.Background(), config,
		hostConfig, nil, containerName,
	)
	if err != nil {
		return "", err
	}

	return created.ID, nil
}

func (cloud *Cloud) FollowLogs(ctx context.Context, container string, send func(string)) error {
	reader, err := cloud.client.ContainerLogs(
		ctx, container, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Tail:       "all",
		},
	)
	if err != nil {
		return err
	}

	defer reader.Close()

	buffer := make([]byte, 1024)
	for {
		size, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		send(string(buffer[:size]))
	}

	return nil
}

func (cloud *Cloud) StartContainer(container string) error {
	err := cloud.client.ContainerStart(
		context.Background(), container,
		types.ContainerStartOptions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (cloud *Cloud) DestroyContainer(container string) error {
	err := cloud.client.ContainerRemove(
		context.Background(), container,
		types.ContainerRemoveOptions{
			Force: true,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (cloud *Cloud) Exec(
	ctx context.Context,
	logger lorg.Logger,
	publish func(string),
	container string,
	command,
	env []string,
) error {
	exec, err := cloud.client.ContainerExecCreate(
		ctx, container,
		types.ExecConfig{
			Cmd:          command,
			Env:          env,
			AttachStdout: true,
			AttachStderr: true,
		},
	)
	if err != nil {
		return err
	}

	response, err := cloud.client.ContainerExecAttach(
		ctx, exec.ID,
		types.ExecStartCheck{},
	)
	if err != nil {
		return err
	}

	writer := &execWriter{logger: logger, publish: publish}

	_, err = stdcopy.StdCopy(writer, writer, response.Reader)
	if err != nil {
		return karma.Format(err, "unable to read stdout of exec/attach")
	}

	return nil
}

func (cloud *Cloud) WriteLogs(
	logsDir, container, packageName string,
) error {
	logfile, err := os.OpenFile(
		filepath.Join(logsDir, packageName),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0o644,
	)
	if err != nil {
		return err
	}

	reader, err := cloud.client.ContainerLogs(
		context.Background(), container, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		},
	)
	if err != nil {
		return err
	}

	defer reader.Close()

	_, err = io.Copy(logfile, reader)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func (cloud *Cloud) Cleanup() error {
	options := types.ContainerListOptions{}

	containers, err := cloud.client.ContainerList(
		context.Background(),
		options,
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to list containers",
		)
	}

	destroyed := 0
	for _, container := range containers {
		if _, ours := container.Labels[ImageLabelKey]; ours {
			infof(
				"cleanup: destroying container %q %q in status: %s",
				container.ID,
				container.Names,
				container.Status,
			)

			err := cloud.DestroyContainer(container.ID)
			if err != nil {
				return karma.Describe("id", container.ID).Format(
					err,
					"unable to destroy container",
				)
			}

			destroyed++
		}
	}

	infof("cleanup: destroyed %d containers", destroyed)

	return nil
}
