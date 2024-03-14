package cryptography

type secureMessage struct {
	Data       string   `json:"data"`
	Signatures []string `json:"signatures,omitempty"`
}

type Cryptography interface {
	// Encrypt message without signatures with a help of public key
	Encrypt(data []byte, key string) ([]byte, error)
	// Decrypt message without signatures
	Decrypt(msg []byte) (interface{}, []string, error)
	// EncryptAndSign encrypt data and signs it, if other signatures was presented in message and you want to add more signatures pass it to func
	EncryptAndSign(data []byte, signatures []string, key string) ([]byte, error)
	// Checks list of signatures on data
	CheckSignatures(data []byte, signatures, publicKeys []string) error
}
