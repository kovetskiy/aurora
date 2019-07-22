package signature

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"strconv"
	"time"

	"github.com/reconquest/karma-go"
)

const (
	SignatureTTL = 30
)

type Signature struct {
	Time int64
	Sign []byte
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

	block, _ := pem.Decode([]byte(pemdata))

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to parse prcks1 private key block",
		)
	}

	return key, nil
}
