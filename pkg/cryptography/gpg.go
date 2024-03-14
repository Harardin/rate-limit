package cryptography

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/errors"
)

type gpgService struct {
	privateKey         string
	privateKeyPassword []byte
}

// NewGPGCryptography Creates new GPG security service if param v nil we take keys from file system
func NewGPGCryptography(privateKeyName, privateKeyPassword string) (Cryptography, error) {
	return &gpgService{
		privateKey:         privateKeyName,
		privateKeyPassword: []byte(privateKeyPassword),
	}, nil
}

// Encrypt Encrypts message without signatures with public gpg key
func (s *gpgService) Encrypt(data []byte, key string) ([]byte, error) {
	dataInBase64 := base64.StdEncoding.EncodeToString(data)

	return json.Marshal(secureMessage{
		Data: dataInBase64,
	})
}

// Decrypt Decrypt message with private key
func (s *gpgService) Decrypt(msg []byte) (interface{}, []string, error) {
	var secMsg secureMessage
	if err := json.Unmarshal(msg, &secMsg); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal secure message: %+v", err)
	}

	dcData, err := base64.StdEncoding.DecodeString(secMsg.Data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode base64 secure message data: %+v", err)
	}

	bytes, err := s.decrypt(dcData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt msg: %+v", err)
	}

	var ans interface{}
	if err := json.Unmarshal(bytes, &ans); err != nil {
		return nil, nil, err
	}

	return ans, secMsg.Signatures, nil
}

// EncryptAndSign Signs message with private key and then encrypts it with public keys for receiver
func (s *gpgService) EncryptAndSign(data []byte, signatures []string, key string) ([]byte, error) {
	signature, err := s.sign(data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign incoming data: %+v", err)
	}

	signs := append(signatures, signature)

	// Encrypting data
	encData, err := s.encrypt(data, key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt incoming data: %+v", err)
	}

	msg := base64.StdEncoding.EncodeToString(encData)

	return json.Marshal(secureMessage{
		Data:       msg,
		Signatures: signs,
	})
}

// CheckSignatures checks if all signatures are valid
func (s *gpgService) CheckSignatures(data []byte, signatures, publicKeys []string) error {
	// checking signatures
	checkSuccess := make(map[int]bool)

	publicKeysOp := make([]openpgp.EntityList, 0)
	signaturesBytes := make([][]byte, 0, len(publicKeys))
	for i, k := range publicKeys {
		pb, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(k))
		if err != nil {
			return fmt.Errorf("failed to read armored key gpg: %+v", err)
		}
		publicKeysOp = append(publicKeysOp, pb)

		checkSuccess[i] = false

		s, err := base64.StdEncoding.DecodeString(signatures[i])
		if err != nil {
			return fmt.Errorf("failed to decode one of signatures from base64")
		}
		signaturesBytes = append(signaturesBytes, s)
	}

	// checking
	for i := range publicKeys {
		for _, sign := range signaturesBytes {
			if _, err := openpgp.CheckArmoredDetachedSignature(publicKeysOp[i], bytes.NewBuffer(data), bytes.NewBuffer(sign)); err != nil {
				if err == errors.ErrUnknownIssuer {
					continue
				}
				return fmt.Errorf("error on one of signatures: %+v", err)
			}
			checkSuccess[i] = true
		}
	}

	// is all signatures checks was successfully
	for _, v := range checkSuccess {
		if !v {
			return fmt.Errorf("invalid signatures")
		}
	}

	return nil
}

// ------------------

func (s *gpgService) encrypt(data interface{}, key string) ([]byte, error) {
	reader := bytes.NewBufferString(key)

	entList, err := openpgp.ReadArmoredKeyRing(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored public gpg key: %+v", err)
	}

	d, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal incoming interface data to bytes: %+v", err)
	}

	// encrypt string
	buf := new(bytes.Buffer)
	w, err := openpgp.Encrypt(buf, entList, nil, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with opengpg err: %+v", err)
	}
	if _, err := w.Write(d); err != nil {
		return nil, fmt.Errorf("failed to write to buffer writer: %+v", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close buffer writer: %+v", err)
	}

	return buf.Bytes(), nil
}

func (s gpgService) decrypt(msg []byte) ([]byte, error) {
	reader := bytes.NewBufferString(s.privateKey)

	entList, err := openpgp.ReadArmoredKeyRing(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored public gpg key: %+v", err)
	}
	entity := entList[0]

	// Get the passphrase and read the private key.
	// Have not touched the encrypted string yet
	entity.PrivateKey.Decrypt(s.privateKeyPassword)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(s.privateKeyPassword)
	}

	// Decrypt it with the contents of the private key
	md, err := openpgp.ReadMessage(bytes.NewBuffer(msg), entList, nil, nil)
	if err != nil {
		return nil, err
	}
	bytes, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (s *gpgService) sign(data []byte) (string, error) {
	reader := bytes.NewBufferString(s.privateKey)

	entList, err := openpgp.ReadArmoredKeyRing(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read armored public gpg key: %+v", err)
	}
	entity := entList[0]

	// Get the passphrase and read the private key.
	// Have not touched the encrypted string yet
	entity.PrivateKey.Decrypt(s.privateKeyPassword)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(s.privateKeyPassword)
	}

	buf := new(bytes.Buffer)
	if err := openpgp.ArmoredDetachSign(buf, entity, bytes.NewReader(data), nil); err != nil {
		return "", fmt.Errorf("failed to make armored detached signature: %+v", err)
	}

	signature := base64.StdEncoding.EncodeToString(buf.Bytes())
	return signature, nil
}
