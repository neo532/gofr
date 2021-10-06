package slices

/*
 * @abstract Uint64
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

// Uint64 is slice of uint64.
type Uint64 []uint64

// Len returns the length of Uint64.
func (s Uint64) Len() int { return len(s) }

// Value returns the value of Uint64.
func (s Uint64) Value(i int) interface{} { return s[i] }

// Append add a new value to Uint64
func (s Uint64) Append(v interface{}) ISlice { return append(s, v.(uint64)) }

// Make makes a new instance.
func (s Uint64) Make() ISlice { return make(Uint64, 0, s.Len()) }

// Less compares two values.
func (s Uint64) Less(i, j int) bool { return s.Value(i).(uint64) > s.Value(j).(uint64) }

// LessValue compare two values in different directory.
func (s Uint64) LessValue(i int, v interface{}) bool { return s.Value(i).(uint64) > v.(uint64) }

// Swap swaps two value's position.
func (s Uint64) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// IsRighType judges the value's type.
func (s Uint64) IsRighType(v interface{}) bool {
	switch v.(type) {
	case uint64:
		return true
	default:
		return false
	}
}
