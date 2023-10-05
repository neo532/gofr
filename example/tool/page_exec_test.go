package tool

import (
	"fmt"
	"testing"

	"github.com/neo532/gofr/tool"
)

func TestPageExec(t *testing.T) {

	arr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	tool.PageExec(int64(len(arr)), 3, func(b, e int64, p int) (err error) {
		fmt.Println(fmt.Sprintf("%s\t:%v", t.Name(), arr[b:e]))
		return
	})
	// [1 2 3] [4 5 6] [7 8 9] [10]
}
