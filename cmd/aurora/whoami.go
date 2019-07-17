package main

import (
	"fmt"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
)

func handleWhoami(opts Options) error {
	client := NewClient(opts.Address)
	signer := NewSigner(opts.Key)

	var response proto.ResponseWhoAmI
	err := client.Call(
		(*rpc.AuthService).WhoAmI,
		proto.RequestWhoAmI{
			Signature: signer.sign(),
		},
		&response,
	)
	if err != nil {
		return err
	}

	if response.Name == "" {
		fmt.Println("Unauthorized")
	} else {
		fmt.Println(response.Name)
	}

	return nil
}
