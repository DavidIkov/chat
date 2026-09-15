package shared_api

const (
	UserNameMinLen     = 4
	UserNameMaxLen     = 16
	UserPasswordMinLen = 4
	UserPasswordMaxLen = 16
)

func ValidateUserName(name string) Error {
	if len(name) < UserNameMinLen {
		return Error{422, "validation_fail", "too small user name"}
	} else if len(name) > UserNameMaxLen {
		return Error{422, "validation_fail", "too big user name"}
	}
	return Error{Code: 0}
}

func ValidateUserPassword(password string) Error {
	if len(password) < UserNameMinLen {
		return Error{422, "validation_fail", "too small user password"}
	} else if len(password) > UserNameMaxLen {
		return Error{422, "validation_fail", "too big user password"}
	}
	return Error{Code: 0}
}
