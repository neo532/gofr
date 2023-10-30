package lib

/*
 * @abstract math operation
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"fmt"
	"math/big"
	"strconv"
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

func Science2Int(science string) (i int64, err error) {
	var f float64
	if _, err = fmt.Sscanf(science, "%e", &f); err != nil {
		return
	}
	s := fmt.Sprintf("%.0f", f)
	i, err = strconv.ParseInt(s, 64, 10)
	return
}
