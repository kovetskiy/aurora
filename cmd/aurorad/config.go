package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/kovetskiy/ko"
	"github.com/reconquest/karma-go"
)

const defaultConfigPath = `/etc/aurora/aurora.conf`

const defaultConfig = `# enable debug messages
debug: true

# enable trace (very debug) messages
trace: true

# listen specified address in web mode
listen: ":80"

# DSN of database to use (mongodb)
database: "mongodb://localhost/aurora"

# directory with ready-to-install packages
repo_dir: "/srv/http/aurora/"

# directory where logs will be stored
logs_dir: "/var/log/aurora/packages/"

# buffer directory for archives
buffer_dir: "/var/aurora/buffer/"

# threads to spawn for queue processing, 0 = num of cpu cores
threads: 0

# instance name (used for following logs)
instance: "$HOSTNAME"

interval:
  # how often should poll queue
  poll: "2s"
  build:
    # rebuild if stuck in processing more than specified time
    status_processing: "30m"
    # rebuild if succeeded more than specified time
    status_success: "30m"
    # rebuild if failed more than specified time
    status_failure: "60m"

timeout:
  # give up building process
  build: "30m"

# image used for building pkgs
base_image: "aurora"

# settings for cleaning up disk space in repository
history:
	# how many different pkgver-pkgrel combination can exist
	versions: 3
	# same version can have different checksums (for whatever reasons)
	builds_per_version: 3

# bus server is an event pubsub system inside of aurorad
bus:
	listen: ":4242"

# dir with authorized RSA public keys
authorized_keys: "/etc/aurora/authorized_keys"
`

type ConfigHistory struct {
	Versions         int `yaml:"versions" required:"true"`
	BuildsPerVersion int `yaml:"builds_per_version" required:"true"`
}

type Config struct {
	Debug bool
	Trace bool

	Instance  string        `yaml:"instance" required:"true"`
	Listen    string        `required:"true"`
	Database  string        `required:"true"`
	RepoDir   string        `yaml:"repo_dir" required:"true"`
	LogsDir   string        `yaml:"logs_dir" required:"true"`
	BufferDir string        `yaml:"buffer_dir" required:"true"`
	Threads   int           `yaml:"threads"`
	BaseImage string        `yaml:"base_image" required:"true"`
	History   ConfigHistory `yaml:"history" required:"true"`

	Bus struct {
		Listen string `yaml:"listen" required:"true"`
	} `required:"true"`

	Interval struct {
		Poll  time.Duration `yaml:"poll" required:"true"`
		Build struct {
			StatusProcessing time.Duration `yaml:"status_processing" required:"true"`
			StatusSuccess    time.Duration `yaml:"status_success" required:"true"`
			StatusFailure    time.Duration `yaml:"status_failure" required:"true"`
		} `required:"true"`
	} `required:"true"`

	Timeout struct {
		Build string `yaml:"build" required:"true"`
	} `required:"true"`

	AuthorizedKeysDir string `yaml:"authorized_keys" required:"true"`
}

func GenerateConfig(path string) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, []byte(defaultConfig), 0600)
	if err != nil {
		return err
	}

	return nil
}

func GetConfig(path string) (*Config, error) {
	var config Config

	err := ko.Load(path, &config, yaml.Unmarshal)
	if os.IsNotExist(err) && path == defaultConfigPath {
		err := GenerateConfig(path)
		if err != nil {
			return nil, karma.Format(
				err,
				"unable to write default config at %s",
				defaultConfigPath,
			)
		}

		err = ko.Load(path, &config, yaml.Unmarshal)
	}

	if config.Instance == "$HOSTNAME" {
		instance, err := os.Hostname()
		if err != nil {
			return nil, karma.Format(err, "unable to get hostname")
		}

		config.Instance = instance
	}

	return &config, err
}
