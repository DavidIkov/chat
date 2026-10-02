"""Concrete api_server client composed from the transport and domain mixins."""

from __future__ import annotations

from chat_client.apiclient.chats import ChatsMixin
from chat_client.apiclient.transport import Transport
from chat_client.apiclient.users import UsersMixin


class ApiClient(UsersMixin, ChatsMixin, Transport):
    """Stateless typed client: one instance serves every server connection.

    Callers pass ``connection.url`` and ``connection.session.token`` explicitly.
    The client holds no per-connection state, so it is safe to share across the
    whole app and to call from worker threads.
    """
