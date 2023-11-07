package cbc

import (
	"github.com/forgoer/openssl"
	"github.com/neo532/gofr/crypt"
	"github.com/neo532/gofr/crypt/encoding/std"
)

type CBC struct {
	padding string
	key     []byte
	iv      []byte
	coding  crypt.IEncoding
}

type opt func(o *CBC)

func WithPadding(padding string) opt {
	return func(o *CBC) {
		o.padding = padding
	}
}
func WithKey(key string) opt {
	return func(o *CBC) {
		o.key = []byte(key)
	}
}
func WithIv(iv string) opt {
	return func(o *CBC) {
		o.iv = []byte(iv)
	}
}
func WithEncoding(coding crypt.IEncoding) opt {
	return func(o *CBC) {
		o.coding = coding
	}
}

func New(opts ...opt) (os *CBC) {
	os = &CBC{
		padding: openssl.PKCS7_PADDING,
		coding:  std.New(),
	}
	for _, fn := range opts {
		fn(os)
	}
	return os
}

func (o *CBC) Encrypt(origin []byte) (encrypt string, err error) {
	if len(origin) == 0 {
		return
	}
	var en []byte
	en, err = openssl.AesCBCEncrypt(origin, o.key, o.iv, o.padding)
	encrypt = o.coding.Encode(en)
	return
}

func (o *CBC) Decrypt(encrypt string) (origin []byte, err error) {
	if encrypt == "" {
		return
	}
	var en []byte
	en, err = o.coding.Decode(encrypt)
	return openssl.AesCBCDecrypt(en, o.key, o.iv, o.padding)
}
