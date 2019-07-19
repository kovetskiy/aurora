package proto

import "time"

type Package struct {
	Name     string        `bson:"name" json:"name"`
	Version  string        `bson:"version" json:"version"`
	Status   PackageStatus `bson:"status" json:"status"`
	Instance string        `bson:"instance" json:"instance"`
	Date     time.Time     `bson:"date" json:"date"`
	Priority int           `bson:"priority" json:"priority"`
}
