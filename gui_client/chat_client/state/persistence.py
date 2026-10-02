"""Translate :class:`~chat_client.state.session.AppState` to/from config JSON.

Kept separate from both :mod:`chat_client.state.session` (which owns runtime
behaviour) and :mod:`chat_client.config.store` (which owns file I/O) so that the
serialisation schema has exactly one home.
"""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

from chat_client.constants import CONFIG_VERSION
from chat_client.models.user import UserSession
from chat_client.state.session import AppState, ServerConnection


def state_to_config(state: AppState) -> dict:
    """Serialise state into the schema described in :mod:`chat_client.config.store`.

    Connections are written in order with their ``id`` and (possibly null)
    ``session``; ``next_id`` is preserved so ids keep increasing after a restart.
    """
    return {
        "version": CONFIG_VERSION,
        "next_id": state.next_id,
        "connections": [
            {
                "id": connection.id,
                "url": connection.url,
                "name": connection.name,
                "session": (
                    connection.session.to_dict()
                    if connection.session is not None
                    else None
                ),
            }
            for connection in state.connections
        ],
    }


def state_from_config(data: dict) -> AppState:
    """Rebuild an :class:`AppState` from parsed config JSON.

    Must tolerate a missing/partial document (return an empty state), unknown
    extra keys (ignore) and a different ``version`` (see
    ``CONFIG_VERSION`` in :mod:`chat_client.constants`). Invalid connection
    entries are skipped rather than aborting startup.
    """
    state = AppState()
    if not isinstance(data, Mapping):
        return state

    raw_connections = data.get("connections")
    if isinstance(raw_connections, list):
        for entry in raw_connections:
            connection = _connection_from_config(entry)
            if connection is not None:
                state.connections.append(connection)

    # Restore the stored counter, but never below the highest connection id so
    # ids minted after a reload cannot collide with an existing connection.
    highest_id = max((connection.id for connection in state.connections), default=0)
    state.next_id = max(_coerce_int(data.get("next_id")) or 0, highest_id)
    return state


def _connection_from_config(entry: Any) -> ServerConnection | None:
    """Build one connection, or ``None`` when the entry is unusable."""
    if not isinstance(entry, Mapping):
        return None
    connection_id = _coerce_int(entry.get("id"))
    url = entry.get("url")
    if connection_id is None or not isinstance(url, str) or not url:
        return None
    name = entry.get("name")
    return ServerConnection(
        id=connection_id,
        url=url,
        name=name if isinstance(name, str) else "",
        session=_session_from_config(entry.get("session")),
    )


def _session_from_config(value: Any) -> UserSession | None:
    """Rebuild a :class:`UserSession` from a stored ``{"uid", "token"}`` mapping."""
    if not isinstance(value, Mapping):
        return None
    uid = _coerce_int(value.get("uid"))
    token = value.get("token")
    if uid is None or not isinstance(token, str) or not token:
        return None
    return UserSession(uid=uid, token=token)


def _coerce_int(value: Any) -> int | None:
    """Coerce a JSON number/string to ``int``; ``None`` when not representable.

    Booleans are rejected even though ``bool`` subclasses ``int`` in Python, so a
    stray ``true`` never becomes id ``1``.
    """
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, str) and value.strip():
        try:
            return int(value)
        except ValueError:
            return None
    return None
