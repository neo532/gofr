/*
 * @abstract math operation
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */
package lib

import (
	"math/big"
)

func Add(left, right int64) int64 {
	var l = big.NewInt(left)
	var r = big.NewInt(right)
	return l.Add(l, r).Int64()
}

func AddF(left, right float64) float64 {
	var l = big.NewFloat(left)
	var r = big.NewFloat(right)
	rst, _ := l.Add(l, r).Float64()
	return rst
}

func Pow(x, y int) int64 {
	bX := big.NewInt(int64(x))
	bR := big.NewInt(int64(x))
	for i := 0; i < y; i++ {
		bR = bR.Mul(bR, bX)
	}
	return bR.Int64()
}
