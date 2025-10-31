package fluentlog

import (
	"github.com/webmafia/fast"
)

func countFmtArgs(format string) int {
	b := fast.StringToBytes(format) // your helper
	idx := 0
	n := 0   // highest argument index used
	cur := 0 // current argument index (fmt’s internal counter)

	for idx < len(b) {
		if b[idx] != '%' {
			idx++
			continue
		}
		idx++
		if idx >= len(b) {
			break
		}

		switch b[idx] {
		case '%':
			idx++ // literal %%
			continue

		case '[':
			// explicit index %[n]
			idx++
			val := 0
			for idx < len(b) {
				c := b[idx]
				if c >= '0' && c <= '9' {
					val = val*10 + int(c-'0')
					idx++
					continue
				}
				if c == ']' {
					idx++
					break
				}
				break
			}
			if val > 0 {
				cur = val
				if cur > n {
					n = cur
				}
			}

		default:
			// implicit verb — use next sequential argument
			cur++
			if cur > n {
				n = cur
			}
			idx++
		}
	}

	return n
}
