package chats

import (
	"errors"
)

var UnknownMessageFilterTypeByUID = errors.New("unknown message filter type by uid")

// InvalidJoinLinkError is returned when a join link token is unknown or expired.
var InvalidJoinLinkError = errors.New("join link is invalid or expired")
