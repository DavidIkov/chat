"""The composed controller: the client's equivalent of the handler layer.

The Go WebUI has one handler per route, each following the same shape: parse the
form, resolve the ``ServerConnection``, call ``apiclient``, then redirect (PRG)
or re-render the page with an error. Here that shape becomes one method per user
action, split across mixins by domain:

* :class:`~chat_client.controller.base.ControllerBase` — navigation + helpers;
* :class:`~chat_client.controller.servers.ServersMixin` — FR-1..5, FR-15;
* :class:`~chat_client.controller.chats.ChatsMixin` — FR-6..8;
* :class:`~chat_client.controller.chat.ChatMixin` — FR-9..12.

* *parse form* → arguments passed by the view;
* *resolve connection* → ``self._signed_in_server(server_id)``;
* *call apiclient* → ``self._run(...)`` (off the main thread);
* *redirect* → ``self.show_*`` (an ``App.navigate`` call);
* *re-render with error* → ``self._show_api_error`` / the view's banner.

All methods run on the Tk main thread; only the ``work`` callable passed to
``TaskRunner`` runs on a worker. Local validation (validator limits, non-empty
fields) happens before scheduling. See ``docs/architecture.md`` § 6 for the full
WebUI-route → method table.
"""

from __future__ import annotations

from chat_client.controller.base import ControllerBase
from chat_client.controller.chat import ChatMixin
from chat_client.controller.chats import ChatsMixin
from chat_client.controller.servers import ServersMixin


class Controller(ServersMixin, ChatsMixin, ChatMixin, ControllerBase):
    """Every user action and every navigation target in one object."""
