package ghttp

import "errors"

/*
 * @abstract variables
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

var (
	// ErrNotSupportType is a type of error that means invaild type.
	ErrNotSupportType error = errors.New("Invaild type,within string,int,int64,uint64,float64,[]string,[]int,[]int64,[]uint64,[]float64!")
	// ErrMustBeStruct is a type of error that means the type must be a struct.
	ErrMustBeStruct error = errors.New("QueryArgs must be a struct or a struct pointer!")
)
