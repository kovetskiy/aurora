package proto

import "time"

type Package struct {
	Name       string        `bson:"name" json:"name"`
	CloneURL   string        `bson:"clone_url" json:"clone_url"`
	Subdir     string        `bson:"subdir" json:"subdir"`
	Version    string        `bson:"version" json:"version"`
	Status     string        `bson:"status" json:"status"`
	Instance   string        `bson:"instance" json:"instance"`
	Date       time.Time     `bson:"date" json:"date"`
	Priority   int           `bson:"priority" json:"priority"`
	Failures   int           `bson:"failures" json:"failures"`
	BuildTime  time.Duration `bson:"build_time" json:"build_time"`
	PkgverTime time.Duration `bson:"pkgver_time" json:"pkgver_time"`
}
