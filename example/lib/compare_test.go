package lib

import (
	"fmt"
	"testing"

	"github.com/neo532/gofr/lib"
)

func TestCompareVersion(t *testing.T) {

	var v1, v2 string

	v1, v2 = "1.2", "1.2.4"
	fmt.Println(fmt.Sprintf("%s\t:%s\t%s:\t%d", t.Name(), v1, v2, lib.CompareVersion(v1, v2)))

	v1, v2 = "2.2", "1.2"
	fmt.Println(fmt.Sprintf("%s\t:%s\t%s:\t%d", t.Name(), v1, v2, lib.CompareVersion(v1, v2)))

	v1, v2 = "2.2", "2.2.0"
	fmt.Println(fmt.Sprintf("%s\t:%s\t%s:\t%d", t.Name(), v1, v2, lib.CompareVersion(v1, v2)))

	v1, v2 = "2.2", "2.2"
	fmt.Println(fmt.Sprintf("%s\t:%s\t%s:\t%d", t.Name(), v1, v2, lib.CompareVersion(v1, v2)))

	v1, v2 = "0.2", "a.2"
	fmt.Println(fmt.Sprintf("%s\t:%s\t%s:\t%d", t.Name(), v1, v2, lib.CompareVersion(v1, v2)))
}
