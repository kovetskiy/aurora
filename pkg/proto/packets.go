package proto

import (
	"github.com/kovetskiy/aurora/pkg/signature"
)

var (
	DefaultBusServerPort = 4242
)

type RequestListPackages struct {
}

type RequestGetPackage struct {
	Name string `json:"name"`
}

type RequestGetLogs struct {
	Name string `json:"name"`
}

type RequestGetBus struct {
	Name string `json:"name"`
}

type RequestAddPackage struct {
	Signature *signature.Signature `json:"signature"`
	Name      string               `json:"name"`
}

type RequestRemovePackage struct {
	Signature *signature.Signature `json:"signature"`
	Name      string               `json:"name"`
}

type ResponseListPackages struct {
	Packages []*Package `json:"packages"`
}

type ResponseGetPackage struct {
	Package *Package `json:"package"`
}

type ResponseGetLogs struct {
	Logs string `json:"logs"`
}

type ResponseGetBus struct {
	Stream string `json:"stream"`
}

type ResponseAddPackage struct{}

type ResponseRemovePackage struct{}

type RequestWhoAmI struct {
	Signature *signature.Signature `json:"signature"`
}

type ResponseWhoAmI struct {
	Name string `json:"name"`
}
