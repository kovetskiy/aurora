package proto

import (
	"fmt"
	"time"
)

type Build struct {
	Package  string        `json:"package,omitempty" bson:"package,omitempty"`
	Status   PackageStatus `json:"status,omitempty" bson:"status,omitempty"`
	Error    error         `json:"error,omitempty" bson:"error,omitempty"`
	Instance string        `json:"instance,omitempty" bson:"instance,omitempty"`
	Archive  string        `json:"archive,omitempty" bson:"archive,omitempty"`
	At       time.Time     `json:"at,omitempty" bson:"at,omitempty"`
}

func (build *Build) String() string {
	return fmt.Sprintf(
		"package=%q status=%q error=%v instance=%q archive=%q at=%v",
		build.Package, build.Status, build.Error,
	)
}
