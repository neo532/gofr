package tool

/*
 * @abstract page exec
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-10-16
 * @demo
	 var arr = []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	 PageExec(len(arr), 3, func(b, e int) {
		 fmt.Println(arr[b:e])
	 })
	 // [1 2 3] [4 5 6] [7 8 9] [10]
*/

import (
	"math"
)

// PageExec make slice execute in paging.
func PageExec(total int, pageSize int, fn func(begin, end int)) {
	if total == 0 || pageSize == 0 {
		return
	}
	pageNum := int(math.Ceil(float64(total) / float64(pageSize)))
	var b, e, i int
	for i = 0; i < pageNum; i++ {

		b = i * pageSize

		e = b + pageSize
		if e > total {
			e = total
		}

		fn(b, e)
	}
	return
}
