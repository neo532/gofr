package gofr

import "testing"

func TestNewAPP(t *testing.T) {

	err := New().Run()

	if err != nil {
		t.Errorf("%s has error[%+v]", t.Name(), err)
	}
}
