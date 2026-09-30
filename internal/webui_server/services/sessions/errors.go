package sessions

import (
	"errors"
)

var ServerNotFoundError = errors.New("server not found")
var InvalidServerURLError = errors.New("server url must be an http or https url")
