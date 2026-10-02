"""Exception hierarchy for the client.

Mirrors the error shapes of the Go WebUI:

* ``apiclient.APIError`` corresponds to ``services/apiclient.APIError``
  (``{status, field, message}`` decoded from ``{"error":{...}}`` or a non-2xx
  response);
* ``ServerNotFoundError`` / ``InvalidServerUrlError`` correspond to the
  ``services/sessions`` sentinels;
* ``ConfigError`` and ``ValidationError`` are client-only additions.

All errors are catchable as :class:`AppError`, which the controller uses as the
single "expected failure" type to route to the active view's error banner.
"""

from __future__ import annotations

__all__ = [
    "AppError",
    "APIError",
    "ConfigError",
    "InvalidServerUrlError",
    "ServerNotFoundError",
    "ValidationError",
]


class AppError(Exception):
    """Base class for every expected client failure."""


class APIError(AppError):
    """A failure reported by an ``api_server``.

    ``field`` names the offending input when the backend reported one (see
    ``internal/shared/api/error.go``); ``status`` is the HTTP status code, if
    the failure came from an actual response.
    """

    def __init__(self, message: str, *, status: int | None = None, field: str = "") -> None:
        super().__init__(message)
        self.status = status
        self.field = field

    def __str__(self) -> str:  # pragma: no cover - trivial
        return self.args[0] if self.args else ""


class ConfigError(AppError):
    """The local config file exists but could not be read or parsed."""


class ServerNotFoundError(AppError):
    """No ``ServerConnection`` with the requested id exists in the session."""

    def __init__(self, server_id: int | None = None) -> None:
        super().__init__("server not found")
        self.server_id = server_id


class InvalidServerUrlError(AppError):
    """A user-supplied api_server URL is not a plain ``http``/``https`` URL."""

    def __init__(self) -> None:
        super().__init__("server url must be an http or https url")


class ValidationError(AppError):
    """Local input validation failed before any backend call was made.

    The WebUI can only validate in Go on the server; a desktop client can check
    the same limits (see ``constants``) instantly and highlight the field.
    """

    def __init__(self, message: str, *, field: str = "") -> None:
        super().__init__(message)
        self.field = field
