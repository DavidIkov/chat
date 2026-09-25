package users

import (
	"errors"
)

var InvalidCredentialsError = errors.New("invalid credentials")
var TokenNotFoundError = errors.New("token not found")
