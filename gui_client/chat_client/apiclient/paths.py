"""api_server endpoint paths and header constants.

Mirrors ``internal/webui_server/services/apiclient/constants.go``. Paths
containing ``%d`` are formatted with an api_server chat uid.
"""

from __future__ import annotations

BEARER_PREFIX = "Bearer "

# --- users ------------------------------------------------------------------

USER_REGISTER = "/user/register"
USER_LOGIN = "/user/login"
USER_LOGOUT = "/user/logout"
USER_DELETE = "/user/delete"
USER_GET = "/user/get"

# --- chats ------------------------------------------------------------------

CHAT_CREATE = "/chat/create"
CHAT_GET_CHATS = "/chat/get_chats"
CHAT_GET_MESSAGES = "/chat/%d/get_messages"
CHAT_SEND_MESSAGE = "/chat/%d/send_message"
CHAT_GET_MEMBERS = "/chat/%d/get_members"
CHAT_CREATE_JOIN_LINK = "/chat/%d/create_join_link"
CHAT_LEAVE = "/chat/%d/leave"
CHAT_JOIN_CHAT = "/chat/join_chat"
