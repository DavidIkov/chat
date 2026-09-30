package validator

import (
	"chat/internal/shared/api"
	"strings"
)

// Validators return the canonical form of the value they validate. Callers MUST
// store the returned string rather than their input, otherwise the same logical
// value could be stored in several forms ("  dave", "dave") and later lookups
// would fail to match it.

func ValidateUserName(name string) (string, *api.Error) {
	name = strings.TrimSpace(name)
	if len(name) < UserNameMinLen {
		return name, errUserNameTooSmall
	} else if len(name) > UserNameMaxLen {
		return name, errUserNameTooBig
	}
	return name, nil
}

func ValidateUserPassword(password string) (string, *api.Error) {
	if len(password) < UserPasswordMinLen {
		return password, errUserPasswordTooSmall
	} else if len(password) > UserPasswordMaxLen {
		return password, errUserPasswordTooBig
	}
	return password, nil
}

func ValidateChatName(name string) (string, *api.Error) {
	name = strings.TrimSpace(name)
	if len(name) < ChatNameMinLen {
		return name, errChatNameTooSmall
	} else if len(name) > ChatNameMaxLen {
		return name, errChatNameTooBig
	}
	return name, nil
}

func ValidateMessageText(text string) (string, *api.Error) {
	text = strings.TrimSpace(text)
	if len(text) < MessageMinLen {
		return text, errMessageTextTooSmall
	} else if len(text) > MessageMaxLen {
		return text, errMessageTextTooBig
	}
	return text, nil
}
