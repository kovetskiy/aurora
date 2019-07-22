package signature

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/reconquest/karma-go"
)

type Keys []Key

type Key struct {
	Signer    *Signer
	PublicKey *rsa.PublicKey
}

func ReadKeys(dir string) (Keys, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to open keys dir",
		)
	}

	keys := []Key{}
	for _, path := range paths {
		name := filepath.Base(path)

		raw, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, karma.Format(
				err,
				"unable to read %q", path,
			)
		}

		block, _ := pem.Decode(raw)
		if block == nil {
			return nil, fmt.Errorf("unable to decode PEM block: %q", path)
		}

		key, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, karma.Format(
				err,
				"unable to parse pkcs1 public key: %q", path,
			)
		}

		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf(
				"rsa public key expected but got %T", key,
			)
		}

		keys = append(keys, Key{
			Signer:    &Signer{Name: name},
			PublicKey: publicKey,
		})
	}

	return keys, nil
}

func (keys Keys) Verify(signature *Signature) *Signer {
	if signature == nil {
		return nil
	}

	for _, key := range keys {
		if err := signature.Verify(key.PublicKey); err == nil {
			return key.Signer
		}
	}

	return nil
}
