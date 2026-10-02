"""Domain value objects shared across the client.

These mirror the JSON contract of ``internal/shared/api``. Only the entities the
client actually renders are modelled; request bodies are plain ``dict`` values
built in ``apiclient``.
"""

from __future__ import annotations

from chat_client.models.chat import Chat, JoinLink, Member, Message
from chat_client.models.shared import TimestampMs, Uid
from chat_client.models.user import User, UserSession

__all__ = [
    "Chat",
    "JoinLink",
    "Member",
    "Message",
    "TimestampMs",
    "Uid",
    "User",
    "UserSession",
]
