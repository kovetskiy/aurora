package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	logger   lorg.Logger

	container string
	address   string
	dir       string
	process   *execution.Operation

	sourcesDir    string
	repositoryDir string

	session runcmd.Runner
	done    map[string]bool
}

func (build *build) String() string {
	return build.pkg.Name
}

func (build *build) Process() {
	build.logger.Infof("processing %s", build.pkg.Name)

	build.pkg.Status = "processing"
	build.pkg.Date = time.Now()

	build.database.set(build.pkg.Name, build.pkg)

	err := saveDatabase(build.database)
	if err != nil {
		build.logger.Error(
			ser.Errorf(
				err, "can't save database with new package data",
			),
		)
		return
	}

	archive, err := build.build()
	if err != nil {
		build.logger.Error(err)

		build.pkg.Status = "error"
	} else {
		build.pkg.Status = "success"
	}

	build.logger.Infof(
		"package %s has been built: %s",
		build.pkg.Name, archive,
	)

	build.database.set(build.pkg.Name, build.pkg)

	err = saveDatabase(build.database)
	if err != nil {
		build.logger.Error(
			ser.Errorf(
				err, "can't save database with new package data",
			),
		)
		return
	}

	build.logger.Infof("database saved")

	build.logger.Info("updating repository")

	err = build.repoadd(archive)
	if err != nil {
		build.logger.Error(
			ser.Errorf(
				err, "can't update aurora repository",
			),
		)
	}

	build.logger.Infof("repository has been updated")
}

func (build *build) repoadd(path string) error {
	cmd := exec.Command(
		"repo-add",
		filepath.Join(build.repositoryDir, "aurora.db.tar"),
		path,
	)

	err := lexec.NewExec(lexec.Loggerf(build.logger.Tracef), cmd).Run()
	if err != nil {
		return err
	}

	return nil
}

func (build *build) build() (string, error) {
	defer build.shutdown()

	err := build.bootstrap()
	if err != nil {
		return "", err
	}

	err = build.connect()
	if err != nil {
		return "", err
	}

	err = build.compile(build.pkg.Name)
	if err != nil {
		return "", ser.Errorf(
			err,
			"can't build package %s", build.pkg.Name,
		)
	}

	archives, err := filepath.Glob(
		filepath.Join(
			build.dir, "aurora", build.pkg.Name, "*.pkg.*",
		),
	)
	if err != nil {
		return "", ser.Errorf(
			err, "can't stat built package archive",
		)
	}

	for _, archive := range archives {
		target := filepath.Join(build.repositoryDir, filepath.Base(archive))

		err = copyFile(archive, target)
		if err != nil {
			return "", ser.Errorf(
				err,
				"can't copy built package archive %s -> %s",
				archive, target,
			)
		}

		return target, nil
	}

	return "", errors.New("built archive file not found")
}

func (build *build) connect() error {
	for retry := 1; retry <= connectionMaxRetries; retry++ {
		build.logger.Debugf(
			"establishing connection to %s:22",
			build.address,
		)

		time.Sleep(time.Millisecond * connectionTimeoutMS)

		var err error
		build.session, err = runcmd.NewRemotePassAuthRunner(
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
	if build.done == nil {
		build.done = map[string]bool{}
	}

	if _, ok := build.done[pkgname]; ok {
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
				err, "can't build package %s (dependency of %s)",
				item, pkgname,
			)
		}
	}

	err = build.makepkg(pkgname)
	if err != nil {
		return err
	}

	build.done[pkgname] = true

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
		build.session.Command(name, arg...),
	)

	stdout, _, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return stdout, nil
}

func extractPackageVersion(archive, base string) string {
	extension := strings.LastIndex(archive, ".pkg")
	if extension < 0 {
		return ""
	}

	// remove prefix with package name, remove suffix .pkg{.tar,.tar.xz,}
	version := archive[len(base+" -"):extension]

	// remove suffix with architecture
	suffix := strings.LastIndex(version, "-")
	if suffix < 0 {
		return version
	}

	version = version[:suffix]

	return version
}

func copyFile(source, target string) (err error) {
	sourceFile, err := os.OpenFile(source, os.O_RDONLY, 0600)
	if err != nil {
		return ser.Errorf(
			err, "can't open %s", source,
		)
	}

	defer sourceFile.Close()

	err = os.MkdirAll(filepath.Dir(target), 0775)
	if err != nil {
		return ser.Errorf(
			err, "can't mkdir %s", filepath.Dir(target),
		)
	}

	targetFile, err := os.OpenFile(
		target, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644,
	)
	if err != nil {
		return ser.Errorf(
			err, "can't open %s", target,
		)
	}

	defer func() {
		closeErr := targetFile.Close()
		if err == nil && closeErr != nil {
			err = ser.Errorf(
				err, "can't close %s", target,
			)
		}
	}()

	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		return ser.Errorf(
			err, "can't copy contents",
		)
	}

	err = targetFile.Sync()
	if err != nil {
		return ser.Errorf(
			err, "can't sync %s", target,
		)
	}

	return nil
}

func chown(path, value string) error {
	command := exec.Command("chown", value, path)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"chown %s %s failed (%s): %s", value, path, err, string(output),
		)
	}

	return nil
}

func chmod(path, value string) error {
	command := exec.Command("chmod", value, path)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"chmod %s %s failed (%s): %s", value, path, err, string(output),
		)
	}

	return nil
}

func cleanupDirectory(directory string) error {
	cleanup := func(path string, info os.FileInfo, err error) error {
		if os.IsNotExist(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if path == directory {
			return nil
		}

		return os.RemoveAll(path)
	}

	return filepath.Walk(directory, cleanup)
}

func fileExists(path ...string) bool {
	_, err := os.Stat(filepath.Join(path...))
	return !os.IsNotExist(err)
}

func hasPrefixURL(url string) bool {
	return strings.Contains(url, "://")
}
