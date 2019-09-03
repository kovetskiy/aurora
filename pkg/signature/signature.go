package signature

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/reconquest/karma-go"
)

const (
	SignatureTTL = 30
)

type Signature struct {
	Time int64  `json:"time"`
	Sign []byte `json:"sign"`
}

func New(key *rsa.PrivateKey) *Signature {
	sign := Signature{}

	sign.Time = time.Now().UnixNano()

	hash := getHash(sign.Time)

	block, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, hash[:], nil)
	if err != nil {
		panic(err)
	}

	sign.Sign = block

	return &sign
}

func getHash(i int64) []byte {
	hash := sha256.Sum256([]byte(strconv.FormatInt(i, 10)))
	return hash[:]
}

func (sign Signature) Verify(key *rsa.PublicKey) error {
	return rsa.VerifyPSS(
		key,
		crypto.SHA256,
		getHash(sign.Time),
		sign.Sign,
		nil,
	)
}

func ReadPrivateKeyFile(path string) (*rsa.PrivateKey, error) {
	pemdata, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParseRawPrivateKey([]byte(pemdata))
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to parse pcks1 private key block",
		)
	}

	switch typed := key.(type) {
	case *rsa.PrivateKey:
		return typed, nil
	default:
		return nil, fmt.Errorf(
			"unsupported type of private key: %T, "+
				"expected to get RSA private key",
			key,
		)
	}
}
