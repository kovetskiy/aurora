package proto

import (
	"github.com/kovetskiy/aurora/pkg/signature"
)

var DefaultBusServerPort = 4242

type RequestListPackages struct {
	Signature *signature.Signature `json:"signature"`
}

type RequestGetPackage struct {
	Signature *signature.Signature `json:"signature"`
	Name      string               `json:"name"`
}

type RequestGetLogs struct {
	Signature *signature.Signature `json:"signature"`
	Name      string               `json:"name"`
}

type RequestGetBus struct {
	Signature *signature.Signature `json:"signature"`
	Name      string               `json:"name"`
}

type RequestAddPackage struct {
	Signature *signature.Signature `json:"signature"`
	Name      string               `json:"name"`
	CloneURL  string               `json:"clone_url,omitempty"`
	Subdir    string               `json:"subdir,omitempty"`
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
