package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/kovetskiy/aur-go"
	"github.com/kovetskiy/lorg"
	"github.com/reconquest/faces/api/hastur"
	"github.com/reconquest/faces/execution"
	"github.com/reconquest/lexec-go"
	"github.com/reconquest/ser-go"
	"github.com/theairkit/runcmd"
)

const (
	connectionMaxRetries = 10
	connectionTimeoutMS  = 500
)

var (
	cloud *hastur.Hastur
)

type build struct {
	database *database
	pkg      pkg
	network  string
	logger   lorg.Logger
	root     string
	files    string
	ssh      runcmd.Runner

	archives  map[string]bool
	container string
	address   string
	dir       string
	process   *execution.Operation
}

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) Process() {
	var err error

	infof("processing %s", build.pkg.Name)

	defer build.shutdown()

	err = build.bootstrap()
	if err != nil {
		build.logger.Error(err)
		return
	}

	err = build.connect()
	if err != nil {
		build.logger.Error(err)
		return
	}

	err = build.compile(build.pkg.Name)
	if err != nil {
		build.logger.Error(
			ser.Errorf(
				err,
				"can't build package %s", build.pkg.Name,
			),
		)
	}

	archives, err := filepath.Glob(
		filepath.Join(
			build.dir, "aurora", build.pkg.Name, "*.pkg.*",
		),
	)
	if err != nil {
		return ser.Errorf(
			err, "can't stat package archive files",
		)
	}

	fmt.Printf("XXXXXX build.go:86: archives: %#v\n", archives)
}

func (build *build) connect() error {
	for retry := 1; retry <= connectionMaxRetries; retry++ {
		build.logger.Debugf(
			"establishing connection to %s:22",
			build.address,
		)

		time.Sleep(time.Millisecond * connectionTimeoutMS)

		var err error
		build.ssh, err = runcmd.NewRemotePassAuthRunner(
			"root", build.address+":22", "",
		)
		if err != nil {
			build.logger.Error(
				ser.Errorf(
					err,
					"can't establish connection to container %s [%s:22]",
					build.container, build.address,
				),
			)

			continue
		}

		return nil
	}

	return fmt.Errorf(
		"can't establish connection to container %s [%s:22]",
		build.container, build.address,
	)
}

func (build *build) compile(pkgname string) error {
	if build.archives == nil {
		build.archives = map[string]bool{}
	}

	if _, ok := build.archives[pkgname]; ok {
		return nil
	}

	build.logger.Debugf("fetching package %s", pkgname)

	err := build.fetch(pkgname)
	if err != nil {
		return ser.Errorf(
			err, "can't fetch package %s", pkgname,
		)
	}

	build.logger.Debugf("retrieving dependencies for %s", pkgname)

	depends, err := build.getAURDepends(pkgname)
	if err != nil {
		return ser.Errorf(
			err, "can't get list of dependencies for package %s",
			pkgname,
		)
	}

	for _, item := range depends {
		build.logger.Debugf("dependency: %s", item)

		err = build.compile(item)
		if err != nil {
			return ser.Errorf(
				err, "can't build dependency package: %s",
			)
		}
	}

	build.makepkg(pkgname)

	build.archives[pkgname] = true

	return nil
}

func (build *build) makepkg(pkgname string) error {
	build.logger.Debugf("executing makepkg %s", pkgname)

	_, err := build.exec(
		"bash", "-c", fmt.Sprintf(
			"cd /aurora/%s/ && makepkg -si --noconfirm",
			pkgname,
		),
	)
	if err != nil {
		return err
	}

	return nil
}

func (build *build) fetch(pkg string) error {
	_, err := build.exec(
		"git", "clone", fmt.Sprintf(
			"https://aur.archlinux.org/%s.git", pkg,
		), fmt.Sprintf(
			"/aurora/%s", pkg,
		),
	)

	return err
}

func (build *build) getAURDepends(pkg string) ([]string, error) {
	output, err := build.exec(
		"bash", "-c",
		fmt.Sprintf(
			`. /aurora/%s/PKGBUILD && echo "${depends[@]}"`,
			pkg,
		),
	)
	if err != nil {
		return nil, ser.Errorf(
			err, "can't source PKGBUILD",
		)
	}

	depends := []string{}
	for _, line := range strings.Split(string(output), "\n") {
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

func (build *build) exec(name string, arg ...string) ([]byte, error) {
	cmd := lexec.New(
		lexec.Loggerf(build.logger.Tracef),
		build.ssh.Command(name, arg...),
	)

	stdout, _, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return stdout, nil
}
