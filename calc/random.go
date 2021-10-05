package calc

/*
 * the randomer
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"math/big"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandFloat returns a random float by input.
func RandFloat(min, max float64) float64 {
	if min >= max || 0 == max {
		return max
	}

	//turn to bigfloat
	bFPrec := big.NewFloat(float64(100))
	bFMin := big.NewFloat(min)
	bFMax := big.NewFloat(max)

	//turn bigfloat to int
	bFMin = bFMin.Mul(bFMin, bFPrec)
	bFMax = bFMax.Mul(bFMax, bFPrec)
	iMin, _ := bFMin.Int64()
	iMax, _ := bFMax.Int64()

	//rand
	//rand.Int63n(max-min) + min
	bIMax := big.NewInt(iMax)
	bIMin := big.NewInt(iMin)
	bIRst := big.NewInt(
		rand.Int63n(
			bIMax.Sub(bIMax, bIMin).Int64(),
		),
	)
	bIRst = bIRst.Add(bIRst, bIMin)

	//turn to float
	bFRst := big.NewFloat(float64(bIRst.Int64()))
	rst, _ := bFRst.Quo(bFRst, bFPrec).Float64()
	return rst
}

// RandInt returns a random int by input.
func RandInt(n int) int {
	return rand.Intn(n)
}
