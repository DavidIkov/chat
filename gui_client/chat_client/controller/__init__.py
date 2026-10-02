"""Application controller, split by domain (see
:mod:`chat_client.controller.controller`)."""

from __future__ import annotations

from chat_client.controller.base import ControllerBase
from chat_client.controller.chat import ChatMixin
from chat_client.controller.chats import ChatsMixin
from chat_client.controller.controller import Controller
from chat_client.controller.servers import ServersMixin

__all__ = [
    "ChatMixin",
    "ChatsMixin",
    "Controller",
    "ControllerBase",
    "ServersMixin",
]
