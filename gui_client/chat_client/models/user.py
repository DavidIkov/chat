"""User-domain value objects, mirroring ``internal/shared/api/user``."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Mapping


@dataclass(frozen=True, slots=True)
class User:
    """A user as returned by ``GET /user/get`` (``{uid, name}``)."""

    uid: int
    name: str

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "User":
        return cls(uid=int(data.get("uid", 0)), name=str(data.get("name", "")))


@dataclass(frozen=True, slots=True)
class UserSession:
    """An api_server session (``{uid, token}``) held for one connection.

    Per FR-14 each :class:`~chat_client.state.session.ServerConnection` owns its
    own ``UserSession``, so two connections may be different users even against
    the same ``api_server``.
    """

    uid: int
    token: str

    @classmethod
    def from_dict(cls, data: Mapping[str, Any]) -> "UserSession":
        return cls(uid=int(data.get("uid", 0)), token=str(data.get("token", "")))

    def to_dict(self) -> dict[str, Any]:
        return {"uid": self.uid, "token": self.token}
