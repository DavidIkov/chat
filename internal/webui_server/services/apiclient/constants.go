package apiclient

// bearerPrefix mirrors api_server/handlers/auth's header format.
const bearerPrefix = "Bearer "

// api_server endpoint paths. Paths containing %d are formatted with an api_server
// chat uid (shared.UID).
const (
	userRegisterPath = "/user/register"
	userLoginPath    = "/user/login"
	userLogoutPath   = "/user/logout"
	userDeletePath   = "/user/delete"
	userGetPath      = "/user/get"

	chatCreatePath         = "/chat/create"
	chatGetChatsPath       = "/chat/get_chats"
	chatGetMessagesPath    = "/chat/%d/get_messages"
	chatSendMessagePath    = "/chat/%d/send_message"
	chatGetMembersPath     = "/chat/%d/get_members"
	chatCreateJoinLinkPath = "/chat/%d/create_join_link"
	chatLeavePath          = "/chat/%d/leave"
	chatJoinChatPath       = "/chat/join_chat"
)
