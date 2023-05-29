package user

import (
	"fmt"
	"net/mail"
	"regexp"
)

func checkName(name string) error {
	if len(name) > 12 && len(name) <= 0 {
		return fmt.Errorf("名稱需介於 1~12 字元")
	}
	return nil
}

func checkPasswordFormat(pwd string) error {
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

func checkEmailFormat(email string) error {
	_, err := mail.ParseAddress(email)
	return err
}
