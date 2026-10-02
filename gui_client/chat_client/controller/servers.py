"""Connection operations — parity with ``internal/webui_server/handlers/servers``.

Covers FR-1..5 and FR-15: adding/removing a connection, logging in/out, and
deleting the account on one connection. Mixed into
:class:`~chat_client.controller.controller.Controller`; every method assumes
``self.state``, ``self.api``, ``self.tasks``, ``self.config`` and the helpers on
:class:`~chat_client.controller.base.ControllerBase` exist.

Each method follows the Go handler shape: resolve the ``ServerConnection`` (the
``serverFromRequest`` guard), validate locally, run the api call off the main
thread via ``_run``, then navigate (the WebUI's ``303`` redirect) or re-render the
page with the backend's message (the WebUI's ``server.html`` ``Error`` field).
"""

from __future__ import annotations

from typing import Any

from chat_client.errors import (
    APIError,
    InvalidServerUrlError,
    ServerNotFoundError,
    ValidationError,
)
from chat_client.models.user import UserSession
from chat_client.state.session import ServerConnection

#: Success banner shown once a connection's account has been deleted (the
#: WebUI's ``?deleted=1``).
DELETED_NOTICE = "The account was deleted on this server."


class ServersMixin:
    """``/servers`` and ``/servers/{id}/*`` operations."""

    # --- helpers -----------------------------------------------------------

    def _connection(self, server_id: int) -> ServerConnection | None:
        """Resolve ``server_id``, navigating to the error view when it is gone.

        Mirrors ``serverFromRequest``: a missing id is not a recoverable form
        error, so it becomes the error page.
        """
        try:
            return self._server(server_id)
        except ServerNotFoundError as exc:
            self.show_error(str(exc))
            return None

    def _form_error(self, error: BaseException) -> None:
        """Report a failed form action without leaving the current view.

        Unlike :meth:`ControllerBase._show_api_error` this never navigates, so a
        rejected login/register/delete keeps the form on screen for a retry and
        shows the backend's message in the page banner (the WebUI re-rendering
        ``server.html`` with its ``Error`` field set).
        """
        if isinstance(error, APIError):
            self.app.show_banner_error(str(error) or "request failed")
            return
        self.show_error(str(error) or error.__class__.__name__)

    def _set_view_loading(self, loading: bool) -> None:
        """Toggle the busy indicator on the mounted view, when there is one.

        Views hand off async work without knowing when it finishes, so the
        controller owns the busy state for the length of a ``_run``.
        """
        view = getattr(self.app, "_view", None)
        if view is not None:
            view.set_loading(loading)

    def _establish_session(self, server_id: int, session: UserSession) -> None:
        """Store a returned session, persist and show the connection page (FR-4)."""
        self.state.set_server_session(server_id, session)
        self._persist()
        self.show_server(server_id)

    # --- operations --------------------------------------------------------

    def add_server(self, url: str, name: str) -> None:
        """Validate and add a connection, then open it (``POST /servers``, FR-1).

        Local only: an empty URL raises :class:`ValidationError` and a bad
        scheme/host raises :class:`InvalidServerUrlError` from ``state.add_server``;
        both are shown in the form banner and nothing is added (FR-1).
        """
        try:
            if not url.strip():
                raise ValidationError("a server URL is required", field="url")
            connection = self.state.add_server(url.strip(), name.strip())
        except (ValidationError, InvalidServerUrlError) as exc:
            self.app.show_banner_error(str(exc))
            return
        self._persist()
        self.show_server(connection.id)

    def remove_server(self, server_id: int) -> None:
        """Drop a connection and return to the landing page (FR-3).

        No api_server call; the dropped token is not invalidated on the backend
        (parity with ``RemoveServerHandler``). Persist afterwards.
        """
        if self._connection(server_id) is None:
            return
        self.state.remove_server(server_id)
        self._persist()
        self.show_servers()

    def log_in(self, server_id: int, name: str, password: str) -> None:
        """``POST /user/login`` then store the session and return to the page (FR-4)."""
        connection = self._connection(server_id)
        if connection is None:
            return
        if not name or not password:
            self.app.show_banner_error("name and password are required")
            return
        self._set_view_loading(True)
        self._run(
            lambda: self.api.log_in(connection.url, name, password),
            lambda session: self._establish_session(connection.id, session),
            on_error=self._form_error,
            on_done=lambda: self._set_view_loading(False),
        )

    def register(self, server_id: int, name: str, password: str) -> None:
        """``POST /user/register`` then store the session (FR-4)."""
        connection = self._connection(server_id)
        if connection is None:
            return
        if not name or not password:
            self.app.show_banner_error("name and password are required")
            return
        self._set_view_loading(True)
        self._run(
            lambda: self.api.register(connection.url, name, password),
            lambda session: self._establish_session(connection.id, session),
            on_error=self._form_error,
            on_done=lambda: self._set_view_loading(False),
        )

    def log_out(self, server_id: int) -> None:
        """``POST /user/logout``, clear the local session, persist (FR-5).

        A backend failure is ignored (the WebUI ignores it too) so the user can
        always sign out locally.
        """
        connection = self._connection(server_id)
        if connection is None:
            return
        session = connection.session

        def finish(_result: Any = None) -> None:
            self.state.clear_server_session(server_id)
            self._persist()
            self.show_server(server_id)

        if session is None:
            finish()
            return

        self._set_view_loading(True)
        self._run(
            lambda: self.api.log_out(connection.url, session.token),
            finish,
            on_error=finish,  # ignore the backend error; still sign out locally
            on_done=lambda: self._set_view_loading(False),
        )

    def delete_account(self, server_id: int, delete_messages: bool) -> None:
        """``POST /user/delete``, clear the session, show the notice (FR-15)."""
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server already checked
            return
        self._set_view_loading(True)

        def on_deleted(_result: Any = None) -> None:
            # The token is dead on the backend, so forget it locally as well.
            self.state.clear_server_session(server_id)
            self._persist()
            self.show_server(server_id, notice=DELETED_NOTICE)

        self._run(
            lambda: self.api.delete_user(
                connection.url, session.token, delete_messages
            ),
            on_deleted,
            on_error=self._form_error,
            on_done=lambda: self._set_view_loading(False),
        )
