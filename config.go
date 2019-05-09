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
repo_dir: "/var/http/aurora/"

# directory where logs will be stored
logs_dir: "/var/log/aurora/packages/"

# buffer directory for archives
buffer_dir: "/var/aurora/buffer/"

# threads to spawn for queue processing, 0 = num of cpu cores
threads: 0

interval:
  poll: "2s"
  build:
    status_processing: "30m"
    status_success: "30m"
    status_failure: "60m"

timeout:
  build: "30m"

base_image: "aurora"

history:
	versions: 3
	builds_per_version: 3
`

type Config struct {
	Debug bool
	Trace bool

	Listen    string `required:"true"`
	Database  string `required:"true"`
	RepoDir   string `yaml:"repo_dir" required:"true"`
	LogsDir   string `yaml:"logs_dir" required:"true"`
	BufferDir string `yaml:"buffer_dir" required:"true"`
	Threads   int    `yaml:"threads"`
	BaseImage string `yaml:"base_image" required:"true"`
	History   struct {
		Versions         int `yaml:"versions" required:"true"`
		BuildsPerVersion int `yaml:"builds_per_version" required:"true"`
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

	return &config, err
}
