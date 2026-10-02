"""Chat-domain value objects, mirroring ``internal/shared/api/chat``.

Field names deliberately match the JSON keys (``chat_uid``, ``message_uid``,
``user_uid`` ...) so the objects can be handed straight to the view widgets.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Mapping

from chat_client.models.shared import timestamp_from_json, uid_from_json


@dataclass(frozen=True, slots=True)
class Chat:
    """One chat (``GET /chat/get_chats``)."""

    chat_uid: int
    name: str
    created_at: int = 0
    creator_user_uid: int = 0

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "Chat":
        return cls(
            chat_uid=uid_from_json(data.get("chat_uid")),
            name=str(data.get("name", "")),
            created_at=timestamp_from_json(data.get("created_at")),
            creator_user_uid=uid_from_json(data.get("creator_user_uid")),
        )


@dataclass(frozen=True, slots=True)
class Message:
    """One message (``GET /chat/{uid}/get_messages``)."""

    message_uid: int
    text: str
    user_uid: int = 0
    chat_uid: int = 0
    created_at: int = 0

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "Message":
        return cls(
            message_uid=uid_from_json(data.get("message_uid")),
            text=str(data.get("text", "")),
            user_uid=uid_from_json(data.get("user_uid")),
            chat_uid=uid_from_json(data.get("chat_uid")),
            created_at=timestamp_from_json(data.get("created_at")),
        )


@dataclass(frozen=True, slots=True)
class Member:
    """One chat member (``GET /chat/{uid}/get_members``)."""

    user_uid: int

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "Member":
        return cls(user_uid=uid_from_json(data.get("user_uid")))


@dataclass(frozen=True, slots=True)
class JoinLink:
    """Result of ``POST /chat/{uid}/create_join_link``.

    ``expires_at`` is ``0`` when the link never expires while the server runs.
    """

    token: str
    expires_at: int = 0

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "JoinLink":
        return cls(
            token=str(data.get("token", "")),
            expires_at=timestamp_from_json(data.get("expires_at")),
        )
