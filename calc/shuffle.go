package calc

/*
 * the shuffle
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2020-09-26
 */

// IShuffle defines the standard of the shuffle.
type IShuffle interface {
	Len() int
	Swap(i, j int)
}

// Shuffle shuffles the data's order.
func Shuffle(data IShuffle) {
	for i := data.Len() - 1; i > 0; i-- {
		data.Swap(i, RandInt(i+1))
	}
}
