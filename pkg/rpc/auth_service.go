package rpc

import (
	"net/http"

	"github.com/kovetskiy/aurora/pkg/proto"
	"github.com/kovetskiy/aurora/pkg/signature"
	"github.com/reconquest/karma-go"
)

type AuthService struct {
	signature.Keys
}

func NewAuthService(authorizedKeysDir string) (*AuthService, error) {
	keys, err := signature.ReadAuthorizedKeys(authorizedKeysDir)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to read authorized keys",
		)
	}

	return &AuthService{
		Keys: keys,
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
