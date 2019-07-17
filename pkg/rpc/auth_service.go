package rpc

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/signature"
	"github.com/reconquest/karma-go"
)

type rsaKey struct {
	signer *signature.Signer
	key    *rsa.PublicKey
}

type AuthService struct {
	keys []rsaKey
}

func NewAuthService(authorizedKeysDir string) (*AuthService, error) {
	paths, err := filepath.Glob(filepath.Join(authorizedKeysDir, "*"))
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to open keys dir",
		)
	}

	keys := []rsaKey{}
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

		key, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, karma.Format(
				err,
				"unable to parse pkcs1 public key: %q", path,
			)
		}

		keys = append(keys, rsaKey{
			signer: &signature.Signer{Name: name},
			key:    key,
		})
	}

	return &AuthService{
		keys: keys,
	}, nil
}

func (service *AuthService) WhoAmI(
	source *http.Request,
	request *proto.RequestWhoAmI,
	response *proto.ResponseWhoAmI,
) error {
	signature := request.Signature
	if signature == nil {
		return nil
	}

	signer := service.Verify(signature)
	if signer == nil {
		return nil
	}

	response.Name = signer.Name

	return nil
}

func (service *AuthService) Verify(signature *signature.Signature) *signature.Signer {
	if signature == nil {
		return nil
	}

	for _, key := range service.keys {
		if err := signature.Verify(key.key); err == nil {
			return key.signer
		}
	}

	return nil
}
