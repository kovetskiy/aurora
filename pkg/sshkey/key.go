package sshkey

import (
	"encoding/pem"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/reconquest/karma-go"
	"github.com/youmark/pkcs8"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type key struct {
	raw        []byte
	block      *pem.Block
	extra      []byte
	private    interface{}
	passphrase []byte
}

func (key *key) validate() error {
	if len(key.extra) != 0 {
		return karma.Format(
			errors.New(string(key.extra)),
			`extra data found in the SSH key`,
		)
	}

	return nil
}

func (key *key) isOpenSSH() bool {
	return key.block.Type == "OPENSSH PRIVATE KEY"
}

func (key *key) isPKCS8() bool {
	return key.block.Type == "ENCRYPTED PRIVATE KEY" ||
		key.block.Type == "PRIVATE KEY"
}

func (key *key) isEncrypted() bool {
	if key.block.Type == "ENCRYPTED PRIVATE KEY" {
		return true
	}

	if strings.Contains(key.block.Headers["Proc-Type"], "ENCRYPTED") {
		return true
	}

	if key.isOpenSSH() {
		_, err := ssh.ParseRawPrivateKey([]byte(key.raw))
		return err != nil
	}

	return false
}

func (key *key) parse() error {
	var err error
	switch {
	case key.isPKCS8():
		key.private, err = pkcs8.ParsePKCS8PrivateKey(
			key.block.Bytes,
			nil,
		)

	default:
		key.private, err = ssh.ParseRawPrivateKey(
			[]byte(key.raw),
		)
	}
	return err
}

func readSSHKey(keyring agent.Agent, path string) error {
	var key key
	var err error

	key.raw, err = ioutil.ReadFile(path)
	if err != nil {
		return karma.Format(
			err,
			`can't read SSH key from file`,
		)
	}

	key.block, key.extra = pem.Decode(key.raw)

	err = key.validate()
	if err != nil {
		return err
	}

	if key.isEncrypted() {
		return karma.Format(
			nil,
			"password protected keys are not supported",
		)
	}

	err = key.parse()
	if err != nil {
		return karma.Format(
			err,
			"unable to parse ssh key",
		)
	}

	keyring.Add(agent.AddedKey{
		PrivateKey: key.private,
		Comment:    "passed by orgalorg",
	})

	return nil
}
