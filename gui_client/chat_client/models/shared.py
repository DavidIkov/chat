"""Shared scalar aliases and JSON coercion helpers.

Mirrors ``internal/shared/types.go``: ``UID`` is a ``uint32`` and ``Time`` is
Unix milliseconds in UTC.
"""

from __future__ import annotations

from typing import Any

#: ``shared.UID`` — a ``uint32`` on the wire (users and chats share the space).
Uid = int
#: ``shared.Time`` — Unix milliseconds in UTC.
TimestampMs = int

UID_MAX = 0xFFFFFFFF


def uid_from_json(value: Any) -> Uid:
    """Coerce a JSON value into a UID, treating missing/empty as ``0``."""
    if value is None or value == "":
        return 0
    return int(value)


def timestamp_from_json(value: Any) -> TimestampMs:
    """Coerce a JSON value into a Unix-milliseconds timestamp (``0`` if unset)."""
    if value is None or value == "":
        return 0
    return int(value)
