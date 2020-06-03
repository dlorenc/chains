package sign

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/openpgp"
)

type Signer struct {
	key *openpgp.Entity
}

func NewSigner() (*Signer, error) {
	if _, err := os.Stat("/etc/config-signing/private"); os.IsNotExist(err) {
		return nil, nil
	}
	privateKey, err := os.Open("/etc/config-signing/private")
	if err != nil {
		return nil, err
	}

	el, err := openpgp.ReadArmoredKeyRing(privateKey)
	if err != nil {
		return nil, err
	}
	s := &Signer{
		key: el[0],
	}

	var passphrase []byte
	if _, err := os.Stat("/etc/config-signing/passphrase"); err == nil {
		passphrase, err = ioutil.ReadFile("/etc/config-signing/passphrase")
		if err != nil {
			return nil, err
		}
		if err := s.key.PrivateKey.Decrypt(passphrase); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Signer) Sign(i interface{}) ([]byte, []byte, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, nil, err
	}
	signature, err := s.SignReader(bytes.NewReader(b))
	return signature, b, err
}

func (s *Signer) SignReader(r io.Reader) ([]byte, error) {
	signature := bytes.Buffer{}
	if err := openpgp.ArmoredDetachSignText(&signature, s.key, r, nil); err != nil {
		return nil, err
	}
	return signature.Bytes(), nil
}
