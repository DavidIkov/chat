package api

import (
	"errors"
	"strings"
)

const (
	UserNameMinLen     = 4
	UserNameMaxLen     = 16
	UserPasswordMinLen = 4
	UserPasswordMaxLen = 16
	ChatNameMinLen     = 4
	ChatNameMaxLen     = 32
	MessageMinLen      = 1
	MessageMaxLen      = 256
)

// Validators return the canonical form of the value they validate. Callers MUST
// store the returned string rather than their input, otherwise the same logical
// value could be stored in several forms ("  dave", "dave") and later lookups
// would fail to match it.

func ValidateUserName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) < UserNameMinLen {
		return name, errors.New("too small user name")
	} else if len(name) > UserNameMaxLen {
		return name, errors.New("too big user name")
	}
	return name, nil
}

func ValidateUserPassword(password string) (string, error) {
	if len(password) < UserPasswordMinLen {
		return password, errors.New("too small user password")
	} else if len(password) > UserPasswordMaxLen {
		return password, errors.New("too big user password")
	}
	return password, nil
}

func ValidateChatName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if len(name) < ChatNameMinLen {
		return name, errors.New("too small chat name")
	} else if len(name) > ChatNameMaxLen {
		return name, errors.New("too big chat name")
	}
	return name, nil
}

func ValidateMessageText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if len(text) < MessageMinLen {
		return text, errors.New("too small message text")
	} else if len(text) > MessageMaxLen {
		return text, errors.New("too big message text")
	}
	return text, nil
}
