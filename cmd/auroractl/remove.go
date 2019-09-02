package main

import (
	"fmt"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/kovetskiy/aurora/pkg/signature"
)

func handleRemove(opts Options) error {
	client := rpc.NewClient(opts.Address)
	signer, err := signature.NewSigner(opts.Key)
	if err != nil {
		return err
	}

	err = client.Call(
		(*rpc.PackageService).RemovePackage,
		proto.RequestRemovePackage{
			Signature: signer.Sign(),
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
