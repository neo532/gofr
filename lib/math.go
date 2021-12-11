package lib

/*
 * @abstract math operation
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"math/big"
)

// Add returns the sum by two int64.
func Add(left, right int64) int64 {
	l := big.NewInt(left)
	r := big.NewInt(right)
	return l.Add(l, r).Int64()
}

// AddF returns the sum by two float64.
func AddF(left, right float64) float64 {
	l := big.NewFloat(left)
	r := big.NewFloat(right)
	rst, _ := l.Add(l, r).Float64()
	return rst
}

// Pow returns a number of int64 after two int64's pow.
func Pow(x, y int) int64 {
	bX := big.NewInt(int64(x))
	bR := big.NewInt(int64(x))
	for i := 0; i < y; i++ {
		bR = bR.Mul(bR, bX)
	}
	return bR.Int64()
}
