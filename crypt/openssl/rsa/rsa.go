package rsa

import (
	"github.com/forgoer/openssl"

	"github.com/neo532/gofr/crypt"
	"github.com/neo532/gofr/crypt/encoding/std"
)

type RSA struct {
	publicKey  []byte
	privateKey []byte
	coding     crypt.IEncoding
}

type opt func(o *RSA)

func WithPublicKey(pub string) opt {
	return func(o *RSA) {
		o.publicKey = []byte(pub)
	}
}
func WithPrivateKey(priv string) opt {
	return func(o *RSA) {
		o.privateKey = []byte(priv)
	}
}
func WithEncoding(coding crypt.IEncoding) opt {
	return func(o *RSA) {
		o.coding = coding
	}
}

func New(opts ...opt) (os *RSA) {
	os = &RSA{
		coding: std.New(),
	}
	for _, fn := range opts {
		fn(os)
	}
	return os
}

func (o *RSA) Encrypt(origin []byte) (encrypt string, err error) {
	var en []byte
	en, err = openssl.RSAEncrypt(origin, o.publicKey)
	encrypt = o.coding.Encode(en)
	return
}

func (o *RSA) Decrypt(encrypt string) (origin []byte, err error) {
	var en []byte
	en, err = o.coding.Decode(encrypt)
	return openssl.RSADecrypt(en, o.privateKey)
}
