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

	build.log.Debugf(
		"container %s has been created",
		build.container,
	)

	build.log.Debugf(
		"obtaining container %s network address",
		build.container,
	)

	err = build.obtainContainerInformation(build.container)
	if err != nil {
		return err
	}

	build.log.Debugf(
		"container network address: %s",
		build.address,
	)

	build.log.Debugf(
		"container rootfs: %s",
		build.dir,
	)

	return nil
}

func (build *build) createContainer() (*execution.Operation, error) {
	build.log.Debugf("creating container %s", build.container)

	container := build.cloud.NewContainer().
		SetSourceDirectory(build.sourcesDir).
		SetName(build.container).
		SetPackages(containerPackages).
		SetCommand(containerStartCommand)

	operation := build.cloud.Start(container)

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
			build.log.Errorf(
				"container's sshd has not started, killing process",
			)
			build.killProcess(operation)
		}
	}()

	build.log.Debugf("waiting for container creating log messages")

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
			build.log.Error(
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
		build.log.Error(
			ser.Errorf(
				err, "can't kill container's sshd process",
			),
		)
		return
	}

	build.log.Debugf("container's sshd process released")
}

func (build *build) shutdown() {
	if build.process != nil {
		build.killProcess(build.process)
	}
}

func (build *build) obtainContainerInformation(name string) error {
	containers, err := build.cloud.Query(name)
	if err != nil {
		return ser.Errorf(
			err, "can't query containers information",
		)
	}

	if len(containers) != 1 {
		return fmt.Errorf(
			"containers cloud engine returned %s items, but must be 1",
			len(containers),
		)
	}

	container := containers[0]
	if container.Status != "active" {
		return fmt.Errorf(
			"container status is '%s', but must be '%s'",
			container.Status, "active",
		)
	}

	if container.Address == "" {
		return fmt.Errorf("container address is empty")
	}

	build.address = strings.Split(container.Address, "/")[0]
	build.dir = container.Root

	return nil
}
