package fastjson

import (
	"unsafe"
)

func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func s2b(s string) (b []byte) {
	// Alias the string's backing array as a read-only byte slice without
	// allocating; the result must not be mutated.
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

const maxStartEndStringLen = 80

func startEndString(s string) string {
	if len(s) <= maxStartEndStringLen {
		return s
	}
	start := s[:40]
	end := s[len(s)-40:]
	return start + "..." + end
}
