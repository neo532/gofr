package ecb

import (
	"fmt"
	"runtime"
	"testing"
)

// 测试工具：https://www.nhooo.com/tool/php/
// echo base64_encode(openssl_encrypt('aaaa','AES-256-ECB','rW@vM2UlXKGe2V%!7@%x5mjclBGT0HGc',OPENSSL_RAW_DATA));
var (
	cnt  = "abcdefghijklmnopqrstuvwxyz1234567890------------"
	dst  = "xp35hO2B+kOVSb3QX07XfnukbY23LfI0xz1QNZqzJ70z2PtRTqz3QreYQV14dvQbxKLe56lK8ATmQ5r+o9UboQ=="
	keyM = []byte("rW@vM2UlXKGh2V%!7@%x5mjclBG40HGc")
)

func TestEncrpyt(t *testing.T) {
	cr := New(
		WithKey(keyM),
	)
	right, err := cr.Encrypt([]byte(cnt))
	if err == nil && right == dst {
		fmt.Println(runtime.Caller(0))
	} else {
		t.Errorf("right, err:\t%+v,%+v", right, err)
	}
}
func TestDescrpyt(t *testing.T) {
	cr := New(
		WithKey(keyM),
	)
	origin, err := cr.Decrypt(dst)
	if err == nil && string(origin) == cnt {
	} else {
		t.Errorf("origin, err:\t%+v,%+v", string(origin), err)
	}
}
