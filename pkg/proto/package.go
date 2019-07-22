package proto

import "time"

type Package struct {
	Name      string        `bson:"name" json:"name"`
	Version   string        `bson:"version" json:"version"`
	Status    PackageStatus `bson:"status" json:"status"`
	UpdatedAt time.Time     `bson:"date" json:"date"`
	Priority  int           `bson:"priority" json:"priority"`
}
