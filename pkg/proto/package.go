package proto

import "time"

type Package struct {
	Name     string    `bson:"name" json:"name"`
	CloneURL string    `bson:"clone_url" json:"clone_url"`
	Version  string    `bson:"version" json:"version"`
	Status   string    `bson:"status" json:"status"`
	Instance string    `bson:"instance" json:"instance"`
	Date     time.Time `bson:"date" json:"date"`
	Priority int       `bson:"priority" json:"priority"`
}
