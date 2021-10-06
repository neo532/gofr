package slices

/*
 * @abstract String
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

// String is slice of string.
type String []string

// Len return the length of String.
func (s String) Len() int { return len(s) }

// Value add a new value to String
func (s String) Value(i int) interface{} { return s[i] }

// Append add a new value to String.
func (s String) Append(v interface{}) ISlice { return append(s, v.(string)) }

// Make makes a new instance.
func (s String) Make() ISlice { return make(String, 0, s.Len()) }

// Less returns if first is less second.
func (s String) Less(i, j int) bool { return s.Value(i).(string) < s.Value(j).(string) }

// LessValue returns if first's Value is less v.
func (s String) LessValue(i int, v interface{}) bool { return s.Value(i).(string) < v.(string) }

// Swap swaps two value's position.
func (s String) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// IsRighType judges the value's type.
func (s String) IsRighType(v interface{}) bool {
	switch v.(type) {
	case string:
		return true
	default:
		return false
	}
}
