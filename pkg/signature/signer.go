package signature

import (
	"crypto/rsa"
)

type Signer struct {
	Name string
	key  *rsa.PrivateKey
}

func (signer Signer) String() string {
	if signer.Name == "" {
		return "<unauthorized>"
	}

	return signer.Name
}

func NewSigner(path string) (*Signer, error) {
	key, err := ReadPrivateKeyFile(path)
	if err != nil {
		return nil, err
	}

	return &Signer{key: key}, nil
}

func (signer *Signer) Sign() *Signature {
	if signer == nil {
		return nil
	}

	return New(signer.key)
}
