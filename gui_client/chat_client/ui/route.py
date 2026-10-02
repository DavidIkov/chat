"""Client-side navigation routes.

A desktop app has no URLs, so the webui_server's ``303 See Other`` redirects
become in-process navigation: a controller resolves a :class:`Route` and the
:class:`~chat_client.ui.app.App` shell swaps the visible view. Route names
mirror the WebUI pages/templates so the two codebases stay easy to compare:

===============  ==========================  ============================
Route name       WebUI page                  WebUI route
===============  ==========================  ============================
``servers``      ``index.html``              ``GET /``
``server``       ``server.html``             ``GET /servers/{server_id}``
``chats``        ``chats.html``              ``GET /servers/{id}/chats``
``chat``         ``chat.html``               ``GET /servers/{id}/chats/{uid}``
``error``        ``error.html``              (any 5xx / not-found)
===============  ==========================  ============================
"""

from __future__ import annotations

from dataclasses import dataclass


class RouteName:
    """String constants naming the selectable views."""

    SERVERS = "servers"
    SERVER = "server"
    CHATS = "chats"
    CHAT = "chat"
    ERROR = "error"


@dataclass(frozen=True)
class Route:
    """A resolved navigation target.

    ``server_id`` is the client-local connection id (the WebUI's ``{server_id}``);
    ``chat_uid`` is an api_server chat uid, always interpreted in the context of
    the enclosing ``server_id``. ``message``/``title`` are only meaningful for
    :data:`RouteName.ERROR`; ``notice`` is a success message carried to a view
    (the WebUI's ``?deleted=1`` banner).
    """

    name: str
    server_id: int | None = None
    chat_uid: int | None = None
    message: str = ""
    title: str = "Error"
    notice: str = ""

    @classmethod
    def servers(cls) -> "Route":
        return cls(name=RouteName.SERVERS)

    @classmethod
    def server(cls, server_id: int, *, notice: str = "") -> "Route":
        return cls(name=RouteName.SERVER, server_id=server_id, notice=notice)

    @classmethod
    def chats(cls, server_id: int) -> "Route":
        return cls(name=RouteName.CHATS, server_id=server_id)

    @classmethod
    def chat(cls, server_id: int, chat_uid: int) -> "Route":
        return cls(name=RouteName.CHAT, server_id=server_id, chat_uid=chat_uid)

    @classmethod
    def error(cls, message: str, *, title: str = "Error") -> "Route":
        return cls(name=RouteName.ERROR, message=message, title=title)
