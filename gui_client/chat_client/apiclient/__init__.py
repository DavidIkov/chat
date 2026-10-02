"""Typed, stateless HTTP client for ``api_server``.

Mirrors ``internal/webui_server/services/apiclient``: the client is stateless
with respect to the app's connections, so every method receives the target
``base_url`` and, when required, the ``token`` explicitly. This keeps it trivial
to reuse and to unit-test.

The concrete :class:`ApiClient` is composed from three cooperating pieces:

* :class:`~chat_client.apiclient.transport.Transport` — JSON-over-HTTP plumbing
  and ``{"error": {...}}`` / non-2xx -> :class:`~chat_client.errors.APIError`
  normalisation;
* :class:`~chat_client.apiclient.users.UsersMixin` — ``/user/*`` methods;
* :class:`~chat_client.apiclient.chats.ChatsMixin` — ``/chat/*`` methods.
"""

from __future__ import annotations

from chat_client.apiclient.client import ApiClient

__all__ = ["ApiClient"]
