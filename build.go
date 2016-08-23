package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kovetskiy/aur-go"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/api/hastur"
	"github.com/reconquest/ser-go"
	"github.com/theairkit/runcmd"
)

var (
	cloud *hastur.Hastur
)

type build struct {
	database  *database
	pkg       pkg
	network   string
	logger    lorg.Logger
	root      string
	files     string
	publicKey []byte
	ssh       runcmd.Runner

	builded map[string]bool
}

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) Process() {
	infof("starting build %s", build.pkg.Name)

	container, address, shutdown, err := build.prepare()
	if err != nil {
		build.logger.Error(err)
		return
	}

	defer shutdown()

	build.logger.Debugf(
		"connecting to container %s (%s:22)",
		container, address,
	)

	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 500)

		build.ssh, err = runcmd.NewRemotePassAuthRunner(
			"root", address+":22", "",
		)
		if err != nil {
			build.logger.Error(
				ser.Errorf(
					err,
					"can't connect to container %s (%s:22) ssh",
					container, address,
				),
			)
		}
	}

	if err != nil {
		return
	}

	build.builded = map[string]bool{}

	build.logger.Debugf("building package %s", build.pkg.Name)

	err = build.compile(container, build.pkg.Name)
	if err != nil {
		build.logger.Error(ser.Errorf(
			err, "can't build package %s", build.pkg.Name,
		))
	}
}

func (build *build) compile(container, pkg string) error {
	if _, ok := build.builded[pkg]; ok {
		return nil
	}

	build.logger.Debugf("fetching package %s", pkg)

	err := build.fetch(container, pkg)
	if err != nil {
		return ser.Errorf(
			err, "can't fetch package '%s'", pkg,
		)
	}

	build.logger.Debugf("retrieving dependencies for %s", pkg)

	depends, err := build.getAURDepends(container, pkg)
	if err != nil {
		return ser.Errorf(
			err, "can't get list of dependencies for package '%s'",
		)
	}

	for _, item := range depends {
		build.logger.Debugf("dependency: %s", item)

		err = build.compile(container, item)
		if err != nil {
			return ser.Errorf(
				err, "can't build dependency package: %s",
			)
		}
	}

	build.logger.Debugf("making package %s", pkg)

	cmd, err := build.ssh.Command(
		fmt.Sprintf(
			"bash -c 'cd /aurora/%s/ && "+
				"makepkg -si --noconfirm'",
			pkg,
		),
	)
	if err != nil {
		return ser.Errorf(
			err, "can't open ssh session",
		)
	}

	_, err = cmd.Run()
	if err != nil {
		return ser.Errorf(
			err, "can't make package %s", pkg,
		)
	}

	build.builded[pkg] = true

	return nil
}

func (build *build) fetch(container, pkg string) error {
	cmd, err := build.ssh.Command(
		fmt.Sprintf(
			"git clone https://aur.archlinux.org/%s.git /aurora/%s",
			pkg, pkg,
		),
	)
	if err != nil {
		return ser.Errorf(
			err, "can't open ssh session",
		)
	}

	_, err = cmd.Run()
	if err != nil {
		return ser.Errorf(
			err, "can't clone remote repository",
		)
	}

	return nil
}

func (build *build) getAURDepends(container, pkg string) ([]string, error) {
	cmd, err := build.ssh.Command(
		fmt.Sprintf(
			`bash -c 'source /aurora/%s/PKGBUILD && echo "${depends[@]}"'`,
			pkg,
		),
	)
	if err != nil {
		return nil, ser.Errorf(
			err, "can't open ssh session",
		)
	}

	buffer, err := cmd.Run()
	if err != nil {
		return nil, err
	}

	depends := []string{}
	for _, line := range buffer {
		items := strings.Fields(line)
		for _, item := range items {
			if item != "" {
				depends = append(
					depends,
					strings.Split(item, ">")[0],
				)
			}
		}
	}

	build.logger.Debugf("dependencies for package %s: %q", pkg, depends)

	names := []string{}
	if len(depends) > 0 {
		packages, err := aur.GetPackages(depends...)
		if err != nil {
			return nil, ser.Errorf(
				err,
				"can't obtain information about packages %q from AUR", depends,
			)
		}

		for name, _ := range packages {
			names = append(names, name)
		}
	}

	build.logger.Debugf("aur dependencies for package %s: %q", pkg, names)

	return names, nil
}
