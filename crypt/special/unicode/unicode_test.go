package unicode

import (
	"fmt"
	"testing"
)

func TestRun(t *testing.T) {
	u := New(WithDelimiter('='))
	origin := "_123中abc国"
	en, err := u.Encrypt([]byte(origin))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(en, err)
	ori, e := u.Decrypt(en)
	fmt.Println(string(ori), e)
	if e != nil {
		t.Error(e)
	}
}
