package signature

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_ReturnsValidSign(t *testing.T) {
	test := assert.New(t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	sign := New(key)

	test.NoError(sign.Verify(&key.PublicKey))
}

func TestSignature_Verify_ReturnsFalseIfCorrupted(t *testing.T) {
	test := assert.New(t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	sign := New(key)

	sign.Time += 1

	test.Error(sign.Verify(&key.PublicKey))
}

func TestSignature_Verify_ReturnsFalseOnEmptyStruct(t *testing.T) {
	test := assert.New(t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	sign := Signature{}

	test.Error(sign.Verify(&key.PublicKey))
}
