package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/kovetskiy/aurora/pkg/config"
	"github.com/kovetskiy/aurora/pkg/log"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/regexputil-go"
)

var (
	reArchiveTime     = `(?P<time>\d+)`
	reArchiveName     = `(?P<name>[a-z0-9][a-z0-9@\._+-]+)`
	reArchiveVer      = `(?P<ver>[a-z0-9_.]+-[0-9]+)`
	reArchiveArch     = `(?P<arch>(i686|x86_64))`
	reArchiveExt      = `(?P<ext>tar(.(gz|bz2|xz|lrz|lzo|sz))?)`
	reArchiveFilename = regexp.MustCompile(`^` + reArchiveTime +
		`\.` + reArchiveName +
		`-` + reArchiveVer +
		`-` + reArchiveArch +
		`\.pkg\.` + reArchiveExt + `$`)
)

type Archive struct {
	Instance string
	Archive  string
	Package  string
}

// TODO: should cleanup globally, not only this specified package
func CleanupRepositoryDirectory(
	directory string,
	pkg string,
	cfg config.StorageHistory,
) error {
	globbed, err := filepath.Glob(
		filepath.Join(
			fmt.Sprintf("%s/*.%s-*-*-*.pkg.*", directory, pkg),
		),
	)
	if err != nil {
		return karma.Format(
			err,
			"unable to glob for packages",
		)
	}

	type archive struct {
		Time     string
		Basename string
	}

	builds := map[string][]archive{}
	for _, fullpath := range globbed {
		basename := filepath.Base(fullpath)

		matches := reArchiveFilename.FindStringSubmatch(basename)

		name := regexputil.Subexp(reArchiveFilename, matches, "name")
		if name != pkg {
			continue
		}

		ver := regexputil.Subexp(reArchiveFilename, matches, "ver")
		time := regexputil.Subexp(reArchiveFilename, matches, "time")

		builds[ver] = append(builds[ver], archive{
			Time:     time,
			Basename: basename,
		})
	}

	versions := []string{}
	for version, _ := range builds {
		versions = append(versions, version)
	}

	trash := []string{}
	if len(versions) > cfg.Versions {
		max := cfg.Versions

		sort.Sort(sort.StringSlice(versions))

		for _, version := range versions[max:] {
			for _, archive := range builds[version] {
				trash = append(trash, archive.Basename)
			}

			delete(builds, version)
		}
	}

	for _, archives := range builds {
		if len(archives) <= cfg.BuildsPerVersion {
			continue
		}

		sort.Slice(archives, func(i, j int) bool {
			return archives[i].Time < archives[j].Time
		})

		for _, archive := range archives[cfg.BuildsPerVersion:] {
			trash = append(trash, archive.Basename)
		}
	}

	for _, archive := range trash {
		fullpath := filepath.Join(directory, archive)

		log.Tracef(nil, "removing old pkg: %s", fullpath)

		err := os.Remove(fullpath)
		if err != nil {
			log.Error(
				karma.Format(
					err,
					"unable to remove old pkg: %s",
					fullpath,
				),
			)
		}
	}

	return nil
}
