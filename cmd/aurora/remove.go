package main

import (
	"fmt"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
)

func handleRemove(opts Options) error {
	client := NewClient(opts.Address)
	signer := NewSigner(opts.Key)

	err := client.Call(
		(*rpc.PackageService).RemovePackage,
		proto.RequestRemovePackage{
			Signature: signer.sign(),
			Name:      opts.Package,
		},
		&proto.ResponseRemovePackage{},
	)
	if err != nil {
		return err
	}

	fmt.Println("package has been removed from the queue")

	return nil
}
