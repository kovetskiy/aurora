package proto

import "regexp"

var (
	rePkgName = regexp.MustCompile(`^[a-z0-9][a-z0-9@\._+-]+$`)
)

func IsValidPackageName(name string) bool {
	return rePkgName.MatchString(name)
}
