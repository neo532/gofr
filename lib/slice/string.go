package lib

/*
 * @abstract String
 * @author liuxiaofeng
 * @mail neo532@126.com
 * @date 2021-10-06
 */

// String
type String []string

// Len
func (s String) Len() int { return len(s) }

// Value
func (s String) Value(i int) interface{} { return a[i] }

// Append
func (s String) Append(v interface{}) ISlice { return append(s, v.(string)) }

// Make
func (s String) Make() ISlice { return make(String, 0, s.Len()) }

// Less
func (s String) Less(i, j int) bool { return s.Value(i).(string) > s.Value(j).(string) }

// LessValue
func (s String) LessValue(i int, v interface{}) bool { return s.Value(i).(string) > v.(string) }

// Swap
func (s String) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// IsRighType
func (s String) IsRighType(v interface{}) bool {
	switch v.(type) {
	case string:
		return true
	default:
		return false
	}
}
