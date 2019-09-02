package main

import (
	"fmt"
	"os"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/rpc"
	"github.com/kovetskiy/aurora/pkg/signature"
)

func handleWhoami(opts Options) error {
	client := rpc.NewClient(opts.Address)
	signer, err := signature.NewSigner(opts.Key)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	var response proto.ResponseWhoAmI
	err = client.Call(
		(*rpc.AuthService).WhoAmI,
		proto.RequestWhoAmI{
			Signature: signer.Sign(),
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
