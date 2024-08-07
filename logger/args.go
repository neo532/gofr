package logger

import (
	"runtime"
	"strings"
)

func GetSourceByFunctionName(begin, end int, allows, denys []string) (file string, line int) {
	if allows == nil || denys == nil {
		return
	}

	var f runtime.Frame
	for i := begin; i < end; i++ {
		pc, _, _, ok := runtime.Caller(i)
		if ok {

			fs := runtime.CallersFrames([]uintptr{pc})
			f, _ = fs.Next()
			//fmt.Println(fmt.Sprintf("%v,%v,%v", i, f.Function, f.Line))

			var flag bool
			for _, d := range denys {
				if strings.HasPrefix(f.Function, d) {
					flag = true
					break
				}
			}
			if flag {
				continue
			}
			for _, d := range allows {
				if strings.HasPrefix(f.Function, d) {
					return f.File, f.Line
				}
			}

		}
	}
	return f.File, f.Line
}

func GetSourceByDepth(depth int) (file string, line int) {
	_, file, line, _ = runtime.Caller(depth)
	return file, line

}
