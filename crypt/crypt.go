package crypt

type ICrypt interface {
	Encrypt(origin []byte) (encrpy string, err error)
	Decrypt(encrpy string) (origin []byte, err error)
}

type IEncoding interface {
	Encode(origin []byte) (code string)
	Decode(code string) (origin []byte, err error)
}
