package tools

// 工具函数，避免nil指针
func GetString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}
func GetBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}
func GetUint64(u *uint64) uint64 {
	if u != nil {
		return *u
	}
	return 0
}
func GetInt64(i *uint64) uint64 {
	if i != nil {
		return *i
	}
	return 0
}

func GetFloat64(f *float64) float64 {
	if f != nil {
		return *f
	}
	return 0
}
