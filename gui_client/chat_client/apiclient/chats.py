"""``/chat/*`` api_server methods, mirroring ``apiclient/chats.go``."""

from __future__ import annotations

from collections.abc import Mapping, Sequence

from chat_client.apiclient import paths
from chat_client.apiclient.transport import as_list, as_uid
from chat_client.constants import MESSAGE_PAGE_LIMIT
from chat_client.models.chat import Chat, JoinLink, Member, Message


class ChatsMixin:
    """Chat/message/membership methods.

    Mixed into :class:`~chat_client.apiclient.client.ApiClient`.
    """

    def create_chat(self, base_url: str, token: str, name: str) -> int:
        """Create a chat. ``POST /chat/create`` -> ``{chat_uid}`` (FR-7)."""
        response = self.request(
            "POST", base_url, paths.CHAT_CREATE, token=token, body={"name": name}
        )
        return as_uid(response, "chat_uid")

    def get_chats(
        self, base_url: str, token: str, uids: Sequence[int] | None = None
    ) -> list[Chat]:
        """List chats. ``GET /chat/get_chats``.

        ``uids=None`` (or empty) must send **no** ``uids`` query: the api_server
        then returns every chat the caller belongs to (FR-6, decision D8). A
        non-empty ``uids`` fetches just those chats, which the chat view uses to
        resolve one chat's name (FR-9). The key is repeated per uid
        (``uids=1&uids=2``), matching ``apiclient/chats.go``.
        """
        query = None
        if uids:
            query = {"uids": [str(uid) for uid in uids]}
        response = self.request(
            "GET", base_url, paths.CHAT_GET_CHATS, query=query, token=token
        )
        return [Chat.from_dict(item) for item in as_list(response, "chats")]

    def get_chat_messages(
        self,
        base_url: str,
        token: str,
        chat_uid: int,
        *,
        limit: int = MESSAGE_PAGE_LIMIT,
        before_message_uid: int = 0,
        after_message_uid: int = 0,
    ) -> list[Message]:
        """Read messages. ``GET /chat/{uid}/get_messages`` (FR-9).

        ``limit == 0`` means "let the api_server choose" (it defaults to 50);
        omit the parameter entirely in that case. Non-zero cursors are optional.
        """
        query: dict[str, str] = {}
        if limit:
            query["limit"] = str(limit)
        if before_message_uid:
            query["before_message_uid"] = str(before_message_uid)
        if after_message_uid:
            query["after_message_uid"] = str(after_message_uid)
        response = self.request(
            "GET",
            base_url,
            paths.CHAT_GET_MESSAGES % chat_uid,
            query=query or None,
            token=token,
        )
        return [Message.from_dict(item) for item in as_list(response, "messages")]

    def send_message(self, base_url: str, token: str, chat_uid: int, text: str) -> int:
        """Post a message. ``POST /chat/{uid}/send_message`` -> ``{message_uid}``."""
        response = self.request(
            "POST",
            base_url,
            paths.CHAT_SEND_MESSAGE % chat_uid,
            token=token,
            body={"text": text},
        )
        return as_uid(response, "message_uid")

    def get_chat_members(self, base_url: str, token: str, chat_uid: int) -> list[Member]:
        """List members. ``GET /chat/{uid}/get_members`` (FR-9)."""
        response = self.request(
            "GET", base_url, paths.CHAT_GET_MEMBERS % chat_uid, token=token
        )
        return [Member.from_dict(item) for item in as_list(response, "members")]

    def create_join_link(
        self,
        base_url: str,
        token: str,
        chat_uid: int,
        *,
        lifetime_seconds: int = 0,
        max_uses: int = 0,
    ) -> JoinLink:
        """Mint a join token. ``POST /chat/{uid}/create_join_link`` (FR-11).

        Both limits are opt-in: ``0`` means "no limit" on the backend.
        """
        response = self.request(
            "POST",
            base_url,
            paths.CHAT_CREATE_JOIN_LINK % chat_uid,
            token=token,
            body={"lifetime_seconds": lifetime_seconds, "max_uses": max_uses},
        )
        return JoinLink.from_dict(response if isinstance(response, Mapping) else {})

    def join_chat(self, base_url: str, token: str, join_token: str) -> int:
        """Join by token. ``POST /chat/join_chat`` -> ``{chat_uid}`` (FR-8)."""
        response = self.request(
            "POST",
            base_url,
            paths.CHAT_JOIN_CHAT,
            token=token,
            body={"token": join_token},
        )
        return as_uid(response, "chat_uid")

    def leave_chat(self, base_url: str, token: str, chat_uid: int) -> None:
        """Leave a chat. ``POST /chat/{uid}/leave`` (FR-12)."""
        self.request("POST", base_url, paths.CHAT_LEAVE % chat_uid, token=token)
