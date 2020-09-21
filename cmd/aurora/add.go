package main

import (
	"fmt"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
)

func handleAdd(opts Options) error {
	client := NewClient(opts.Address)
	signer := NewSigner(opts.Key)

	err := client.Call(
		(*rpc.PackageService).AddPackage,
		proto.RequestAddPackage{
			Signature: signer.sign(),
			Name:      opts.Package,
			CloneURL:  opts.CloneURL,
			Subdir:    opts.Subdir,
		},
		&proto.ResponseAddPackage{},
	)
	if err != nil {
		return err
	}

	fmt.Println("Package has been queued")

	return nil
}
