package resp

func ParseInt(b []byte) (int64, bool) {
	var n int64
	var neg bool
	if len(b) == 0 {
		return 0, false
	}
	for i, c := range b {
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int64(c-'0')
	}
	if neg {
		n = -n
	}
	return n, true
}
