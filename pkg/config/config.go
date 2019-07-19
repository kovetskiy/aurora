package config

import (
	"time"

	"github.com/go-yaml/yaml"
	"github.com/kovetskiy/ko"
)

type Log struct {
	Debug bool `yaml:"debug"`
	Trace bool `yaml:"trace"`
}

type RPC struct {
	Log
	Listen            string `yaml:"listen" required:"true"`
	Database          string `yaml:"database" required:"true"`
	Bus               string `yaml:"bus" required:"true"`
	AuthorizedKeysDir string `yaml:"authorized_keys" required:"true"`
}

type Bus struct {
	Log
	Listen            string `yaml:"listen" required:"true"`
	AuthorizedKeysDir string `yaml:"authorized_keys" required:"true"`
}

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

//func GenerateConfig(path string) error {
//    err := os.MkdirAll(filepath.Dir(path), 0755)
//    if err != nil {
//        return err
//    }

//    err = ioutil.WriteFile(path, []byte(defaultConfig), 0600)
//    if err != nil {
//        return err
//    }

//    return nil
//}

func GetRPC(path string) (*RPC, error) {
	var config RPC
	err := ko.Load(path, &config, yaml.Unmarshal)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

//func Load(path string) (*Config, error) {
//    var config Config

//    err := ko.Load(path, &config, yaml.Unmarshal)
//    if os.IsNotExist(err) && path == defaultConfigPath {
//        err := GenerateConfig(path)
//        if err != nil {
//            return nil, karma.Format(
//                err,
//                "unable to write default config at %s",
//                defaultConfigPath,
//            )
//        }

//        err = ko.Load(path, &config, yaml.Unmarshal)
//    }

//    if config.Instance == "$HOSTNAME" {
//        instance, err := os.Hostname()
//        if err != nil {
//            return nil, karma.Format(err, "unable to get hostname")
//        }

//        config.Instance = instance
//    }

//    return &config, err
//}
