package chats

import (
	"errors"
)

var UnknownMessageFilterTypeByUID = errors.New("unknown message filter type by uid")

var InvalidJoinLinkError = errors.New("join link is invalid or expired")
