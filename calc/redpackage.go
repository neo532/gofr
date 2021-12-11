package calc

/*
 * the randomer
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

import (
	"math"
	"math/big"
)

// RedPkgList return a array of float64 by money and count.
func RedPkgList(maxMoney float64, maxCount int) []float64 {
	if maxMoney < float64(maxCount) {
		return nil
	}

	bDoneMoney := big.NewFloat(float64(0))

	bMinMoney := big.NewFloat(float64(0.02))
	bMaxCount := big.NewFloat(float64(maxCount))
	bMaxMoney := big.NewFloat(maxMoney)

	n10 := math.Pow10(2)
	list := make([]float64, 0, maxCount)

	for i := 0; i < maxCount; i++ {
		//remain := maxMoney - doneMoney - (maxCount-i+1)*minMoney
		bRemainCount := big.NewFloat(float64(0))
		bRemainCount = bRemainCount.Sub(bMaxCount, big.NewFloat(float64(1+i)))

		bRmainMoney := big.NewFloat(float64(0))
		bRmainMoney = bRmainMoney.Mul(bRemainCount, bMinMoney)

		bRemain := big.NewFloat(float64(0))
		bRemain = bRemain.Sub(bMaxMoney, bDoneMoney)
		bRemain = bRemain.Sub(bRemain, bRmainMoney)

		bMoney := big.NewFloat(float64(0))
		if i < maxCount-1 {
			bRemain = bRemain.Quo(bRemain, bRemainCount)
			bRemain = bRemain.Quo(bRemain, big.NewFloat(float64(2)))

			remain, _ := bRemain.Float64()
			bMoney = big.NewFloat(RandFloat(0, remain))
			bMoney = bMoney.Add(bMoney, bMinMoney)
		} else {
			bMoney = bMoney.Sub(bMaxMoney, bDoneMoney)
		}
		rst, _ := bMoney.Float64()
		rst2prec := math.Trunc((rst+0.5/n10)*n10) / n10
		list = append(list, rst2prec)

		bDoneMoney = bDoneMoney.Add(bDoneMoney, big.NewFloat(rst2prec))
	}
	return list
}
