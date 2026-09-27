package shared_api

import (
	"errors"
)

const (
	UserNameMinLen     = 4
	UserNameMaxLen     = 16
	UserPasswordMinLen = 4
	UserPasswordMaxLen = 16
)

func ValidateUserName(name string) error {
	if len(name) < UserNameMinLen {
		return errors.New("too small user name")
	} else if len(name) > UserNameMaxLen {
		return errors.New("too big user name")
	}
	return nil
}

func ValidateUserPassword(password string) error {
	if len(password) < UserPasswordMinLen {
		return errors.New("too small user password")
	} else if len(password) > UserPasswordMaxLen {
		return errors.New("too big user password")
	}
	return nil
}
