package ghttp

import "errors"

/*
 * @abstract variables
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

var (
	E_NOT_SUPPORT_TYPE error = errors.New("Invaild type,within string,int,int64,uint64,float64,[]string,[]int,[]int64,[]uint64,[]float64!")
	E_MUST_BE_STRUCT   error = errors.New("QueryArgs must be a struct or a struct pointer!")
)
