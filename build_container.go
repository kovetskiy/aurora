package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/reconquest/faces/execution"
	"github.com/reconquest/ser-go"
)

var (
	containerStartCommand = "" +
		"/bin/sed '/EUID == 0/,/fi/{/exit/d}' -i /bin/makepkg && " +
		"/bin/passwd -d root && " +
		"/bin/ssh-keygen -A && " +
		"/bin/echo 'spawn sshd' && " +
		"/bin/sshd -D"

	containerStartDeadline = time.Minute * 5
	containerPackages      = []string{
		"bash", "coreutils", "git", "bzr", "grep", "sed", "awk",
		"util-linux", "diffutils", "bind-tools",
		"openssh", "dhcpcd", "iproute2", "pacman", "iputils",
		"gzip", "binutils", "sudo", "gcc", "file", "libarchive",
		"pkg-config", "make", "fakeroot",
	}
)

func (build *build) bootstrap() error {
	var err error

	build.container = build.pkg.Name + "-" + fmt.Sprint(time.Now().Unix())

	build.process, err = build.createContainer()
	if err != nil {
		return ser.Errorf(
			err, "can't create container for building package",
		)
	}

	build.logger.Debugf(
		"container %s has been created",
		build.container,
	)

	build.logger.Debugf(
		"obtaining container %s network address",
		build.container,
	)

	err := build.queryContainer()
	if err != nil {
		return err
	}

	build.logger.Debugf(
		"container network address: %s",
		build.address,
	)

	build.logger.Debugf(
		"container rootfs: %s",
		build.dir,
	)

	return nil
}

func (build *build) createContainer() (*execution.Operation, error) {
	build.logger.Debugf("creating container %s", build.container)

	container := cloud.NewContainer().
		SetSourceDirectory(build.files).
		SetName(build.container).
		SetPackages(containerPackages).
		SetCommand(containerStartCommand)

	operation := cloud.Start(container)

	pipe, writer := io.Pipe()
	operation.SetStdout(writer)

	err := operation.Start()
	if err != nil {
		return nil, ser.Errorf(
			err, "can't start container creating process",
		)
	}

	var listening bool
	var done bool

	go func() {
		time.Sleep(containerStartDeadline)

		if !listening {
			done = true
			build.logger.Errorf(
				"container's sshd has not started, killing process",
			)
			build.killProcess(operation)
		}
	}()

	build.logger.Debugf("waiting for container creating log messages")

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, "spawn sshd") {
			listening = true
			break
		}
	}

	if !listening {
		return nil, errors.New("could not spawn sshd process")
	}

	go func() {
		err = operation.Wait()
		if err != nil {
			build.logger.Error(
				ser.Errorf(
					err, "container's sshd has crashed",
				),
			)
		}
	}()

	return operation, nil
}

func (build *build) killProcess(operation *execution.Operation) {
	err := operation.Kill()
	if err != nil {
		build.logger.Error(
			ser.Errorf(
				err, "can't kill container's sshd process",
			),
		)
		return
	}

	build.logger.Debugf("container's sshd process released")
}

func (build *build) shutdown() {
	if build.process != nil {
		build.killProcess(build.process)
	}
}

func (build *build) queryContainer(name string) error {
	containers, err := cloud.Query(name)
	if err != nil {
		return "", ser.Errorf(
			err, "can't query containers information",
		)
	}

	if len(containers) != 1 {
		return "", fmt.Errorf(
			"containers cloud engine returned %s items, but must be 1",
			len(containers),
		)
	}

	container := containers[0]
	if container.Status != "active" {
		return "", fmt.Errorf(
			"container status is '%s', but must be '%s'",
			container.Status, "active",
		)
	}

	if container.Address == "" {
		return "", fmt.Errorf("container address is empty")
	}

	build.address = strings.Split(container.Address, "/")[0]
	build.dir = container.Root

	return nil
}
