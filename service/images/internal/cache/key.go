package cache

import (
	"crypto/md5"
	"encoding/hex"
)

// NewKey returns cache key
func NewKey(src string) string {

	hash := md5.Sum([]byte(src))

	buf := make([]byte, len(hash)*2)
	hex.Encode(buf, hash[:])

	return string(buf)
}
