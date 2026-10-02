"""UI views: one class per :class:`~chat_client.ui.route.RouteName`."""

from __future__ import annotations

from chat_client.ui.views.base import BaseView
from chat_client.ui.views.chat import ChatView
from chat_client.ui.views.chats import ChatsView
from chat_client.ui.views.error import ErrorView
from chat_client.ui.views.server import ServerView
from chat_client.ui.views.servers import ServersView

__all__ = [
    "BaseView",
    "ChatView",
    "ChatsView",
    "ErrorView",
    "ServerView",
    "ServersView",
]
