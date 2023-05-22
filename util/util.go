package util

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"
)

func PasswordHash(pwd string, salt string) string {
	pwd += salt
	return fmt.Sprintf("%x", sha256.Sum256([]byte(pwd)))
}

func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789~@!"
	var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
