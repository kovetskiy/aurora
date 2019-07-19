package proto

import (
	"fmt"
	"time"
)

type Build struct {
	Package  string        `json:"package,omitempty"`
	Status   PackageStatus `json:"status,omitempty"`
	Error    error         `json:"error,omitempty"`
	Instance string        `json:"instance,omitempty"`
	Archive  string        `json:"archive,omitempty"`
	At       time.Time     `json:"at,omitempty"`
}

func (build *Build) String() string {
	return fmt.Sprintf(
		"package=%q status=%q error=%v instance=%q archive=%q at=%v",
		build.Package, build.Status, build.Error,
	)
}

type BuildLogChunk struct {
	Package string `json:"package"`
	Data    string `json:"data"`
}
