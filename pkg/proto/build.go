package proto

import (
	"fmt"
	"time"

	"github.com/reconquest/karma-go"
)

type Build struct {
	Package  string        `json:"package,omitempty" bson:"package,omitempty"`
	Status   PackageStatus `json:"status,omitempty" bson:"status,omitempty"`
	Error    string        `json:"error,omitempty" bson:"error,omitempty"`
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

func (build *Build) Describe() *karma.Context {
	return karma.
		Describe("package", build.Package).
		Describe("status", build.Status).
		Describe("error", build.Error).
		Describe("instance", build.Instance).
		Describe("archive", build.Archive).
		Describe("at", build.At.Format(time.RFC3339))
}
