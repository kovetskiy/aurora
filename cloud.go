package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Cloud struct {
	client *client.Client
}

func (cloud *Cloud) CreateContainer(
	repositoryDir string, containerName string, packageName string,
) (string, error) {
	config := &container.Config{
		Image: "aurora",
		Tty:   true,
		Env: []string{
			fmt.Sprintf("AURORA_PACKAGE=%s", packageName),
		},
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/aurora", repositoryDir),
		},
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

func (cloud *Cloud) WaitContainer(name string) {
	ctx, _ := context.WithTimeout(context.Background(), time.Minute * 30)

	wait, _ := cloud.client.ContainerWait(
		ctx, name,
		container.WaitConditionNotRunning,
	)

	select {
	case <-wait:
		break
	case <-ctx.Done():
		break
	}
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

func (cloud *Cloud) Query(container string) ([]interface{}, error) {
	result := []interface{}{}
	return result, nil
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

func (cloud *Cloud) Exec(container string, command []string) error {
	exec, err := cloud.client.ContainerExecCreate(
		context.Background(), container,
		types.ExecConfig{
			Cmd: command,
		},
	)
	if err != nil {
		return err
	}

	err = cloud.client.ContainerExecStart(
		context.Background(), exec.ID,
		types.ExecStartCheck{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (cloud *Cloud) WriteLogs(
	logsDir string, container string, packageName string,
) error {
	logfile, err := os.OpenFile(
		filepath.Join(logsDir, packageName),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
		0644,
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

	_, err = io.Copy(logfile, reader)
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
