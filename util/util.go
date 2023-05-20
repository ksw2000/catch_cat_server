package util

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

func PasswordFormatChecker(pwd string) error {
	match, err := regexp.MatchString("^[a-zA-Z0-9_@]{8,40}$", pwd)
	if err != nil || !match {
		return fmt.Errorf("密碼僅接受「英文字母、數字、-、_、@」且介於 8 到 40 字元")
	}

	match, err = regexp.MatchString("^.*?\\d+.*?$", pwd)
	if err != nil || !match {
		return fmt.Errorf("密碼必需含有數字")
	}

	match, err = regexp.MatchString("^.*?[a-zA-Z]+.*?$", pwd)
	if err != nil || !match {
		return fmt.Errorf("密碼必需含有英文字母")
	}
	return nil
}

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
