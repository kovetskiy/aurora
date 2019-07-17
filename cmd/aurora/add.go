package main

import (
	"fmt"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
)

func handleAdd(opts Options) error {
	client := NewClient(opts.Address)

	err := client.Call(
		(*rpc.PackageService).AddPackage,
		proto.RequestAddPackage{
			Name: opts.Package,
		},
		&proto.ResponseAddPackage{},
	)
	if err != nil {
		return err
	}

	fmt.Println("package has been queued")

	return nil
}
