package main

import (
	"crypto/rsa"
	"fmt"
	"os"

	"github.com/kovetskiy/aurora/pkg/signature"
)

type signer struct {
	key *rsa.PrivateKey
}

func NewSigner(path string) *signer {
	key, err := signature.ReadPrivateKeyFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		panic(fmt.Errorf("unable to read key: %s", path))
	}

	return &signer{key: key}
}

func (signer *signer) sign() *signature.Signature {
	if signer == nil {
		return nil
	}

	return signature.New(signer.key)
}
