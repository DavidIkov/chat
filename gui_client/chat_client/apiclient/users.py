"""``/user/*`` api_server methods, mirroring ``apiclient/users.go``."""

from __future__ import annotations

from collections.abc import Mapping, Sequence
from typing import Any

from chat_client.apiclient import paths
from chat_client.apiclient.transport import as_list
from chat_client.models.user import User, UserSession


def _user_session(response: Any) -> UserSession:
    """Extract ``resp["user_session"]`` from a register/login response.

    The transport has already turned an ``{"error": ...}`` payload into an
    :class:`~chat_client.errors.APIError`, so here we only decode the success
    shape and tolerate a missing/null session.
    """
    data = response.get("user_session") if isinstance(response, Mapping) else None
    return UserSession.from_dict(data if isinstance(data, Mapping) else {})


class UsersMixin:
    """Account methods. Mixed into :class:`~chat_client.apiclient.client.ApiClient`."""

    def register(self, base_url: str, name: str, password: str) -> UserSession:
        """Create an account. ``POST /user/register`` -> ``{user_session}``.

        Raises :class:`~chat_client.errors.APIError` on rejection (e.g. a name
        already taken, or a validator error with ``field="name"``).
        """
        response = self.request(
            "POST",
            base_url,
            paths.USER_REGISTER,
            body={"name": name, "password": password},
        )
        return _user_session(response)

    def log_in(self, base_url: str, name: str, password: str) -> UserSession:
        """Sign in. ``POST /user/login`` -> ``{user_session}``."""
        response = self.request(
            "POST",
            base_url,
            paths.USER_LOGIN,
            body={"name": name, "password": password},
        )
        return _user_session(response)

    def log_out(self, base_url: str, token: str) -> None:
        """Invalidate the token on the backend. ``POST /user/logout``."""
        self.request("POST", base_url, paths.USER_LOGOUT, token=token)

    def delete_user(self, base_url: str, token: str, delete_messages: bool) -> None:
        """Permanently delete the account. ``POST /user/delete``.

        ``delete_messages`` controls whether the account's messages are also
        removed from every chat (FR-15).
        """
        self.request(
            "POST",
            base_url,
            paths.USER_DELETE,
            token=token,
            body={"delete_messages": bool(delete_messages)},
        )

    def get_users(self, base_url: str, token: str, uids: Sequence[int]) -> list[User]:
        """Resolve uids to display names. ``GET /user/get?uids=1&uids=2``.

        Used by the chat view to turn ``user_uid`` values into author/member
        names (FR-9). ``uids`` is sent by repeating the ``uids`` query key — the
        api_server binds ``[]uint`` from repeated form values (a comma-separated
        single value is rejected), matching ``apiclient/users.go``.
        """
        values = [str(uid) for uid in uids]
        if not values:
            return []
        response = self.request(
            "GET",
            base_url,
            paths.USER_GET,
            query={"uids": values},
            token=token,
        )
        return [User.from_dict(item) for item in as_list(response, "users")]
