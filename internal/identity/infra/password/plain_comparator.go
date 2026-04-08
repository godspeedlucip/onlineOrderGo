package password

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"
)

type MD5Comparator struct{}

func NewMD5Comparator() *MD5Comparator { return &MD5Comparator{} }

// NewPlainComparator keeps backward compatibility with previous wiring.
func NewPlainComparator() *MD5Comparator { return &MD5Comparator{} }

func (p *MD5Comparator) Compare(hashed string, plain string) error {
	if strings.TrimSpace(hashed) == "" || strings.TrimSpace(plain) == "" {
		return errors.New("password mismatch")
	}
	digest := md5.Sum([]byte(plain))
	encoded := hex.EncodeToString(digest[:])
	if !strings.EqualFold(hashed, encoded) {
		return errors.New("password mismatch")
	}
	return nil
}

func HashMD5(plain string) string {
	digest := md5.Sum([]byte(plain))
	return hex.EncodeToString(digest[:])
}
