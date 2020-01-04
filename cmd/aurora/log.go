package main

import (
	"fmt"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
)

func handleLog(opts Options) error {
	client := NewClient(opts.Address)
	signer := NewSigner(opts.Key)

	var response proto.ResponseGetLogs
	err := client.Call(
		(*rpc.PackageService).GetLogs,
		proto.RequestGetLogs{
			Signature: signer.sign(),
			Name:      opts.Package,
		},
		&response,
	)
	if err != nil {
		return err
	}

	fmt.Println(response.Logs)

	return nil
}
