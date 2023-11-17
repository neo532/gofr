package unicode

import (
	"fmt"
	"testing"
)

func TestRun(t *testing.T) {
	var err error
	var en string
	var ori []byte

	origin := "壹_123贰abc叁肆1五1ABC"
	fmt.Println(fmt.Sprintf("origin:\t%+v", origin))

	u := New(WithDelimiter("=="))
	en, err = u.Encrypt([]byte(origin))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(fmt.Sprintf("en:\t%+v", en))

	ori, e := u.Decrypt(en)
	if e != nil {
		t.Error(e)
	}
	fmt.Println(fmt.Sprintf("string(ori):\t%+v", string(ori)))

	if string(ori) != (origin) {
		t.Errorf("not match")
	}
}
