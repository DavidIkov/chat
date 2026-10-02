"""Shared controller infrastructure: navigation and cross-cutting helpers.

The controller is split the same way the Go handler layer is:

* :mod:`chat_client.controller.base` — ``ControllerBase`` (this file): the
  equivalent of ``middleware`` + ``render`` (navigation) plus the shared
  "resolve connection / run a task / report an error" helpers every operation
  needs;
* :mod:`chat_client.controller.servers` — ``/servers/*`` (≈ ``handlers/servers``);
* :mod:`chat_client.controller.chats` — ``…/chats`` list actions;
* :mod:`chat_client.controller.chat` — one chat's page actions;
* :mod:`chat_client.controller.controller` — the composed ``Controller``.

Every method here and in the mixins runs on the Tk main thread. Blocking work is
always handed to :class:`~chat_client.tasks.runner.TaskRunner`.

See ``gui_client/docs/architecture.md`` § 6 for the WebUI-route → method table.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, Callable

from chat_client.apiclient.client import ApiClient
from chat_client.config.store import ConfigStore
from chat_client.errors import APIError, ConfigError
from chat_client.state.persistence import state_to_config
from chat_client.state.session import AppState, ServerConnection
from chat_client.tasks.runner import TaskRunner
from chat_client.ui import dialogs
from chat_client.ui.route import Route, RouteName

if TYPE_CHECKING:  # pragma: no cover
    from chat_client.ui.app import App


class ControllerBase:
    """Navigation plus the helpers shared by every operation mixin."""

    def __init__(
        self,
        app: "App",
        state: AppState,
        api: ApiClient,
        tasks: TaskRunner,
        config: ConfigStore,
    ) -> None:
        self.app = app
        self.state = state
        self.api = api
        self.tasks = tasks
        self.config = config

    # --- navigation (WebUI "redirects") ------------------------------------

    def show_servers(self) -> None:
        """Navigate to the landing page (``GET /``)."""
        self.app.navigate(Route.servers())

    def show_server(self, server_id: int, *, notice: str = "") -> None:
        """Navigate to one connection's page (``GET /servers/{server_id}``)."""
        self.app.navigate(Route.server(server_id, notice=notice))

    def show_chats(self, server_id: int) -> None:
        """Navigate to the chat list (``GET /servers/{server_id}/chats``)."""
        self.app.navigate(Route.chats(server_id))

    def show_chat(self, server_id: int, chat_uid: int) -> None:
        """Navigate to one chat (``GET /servers/{id}/chats/{chat_uid}``)."""
        self.app.navigate(Route.chat(server_id, chat_uid))

    def show_error(self, message: str, *, title: str = "Error") -> None:
        """Show the error page (``error.html``) for an unexpected failure."""
        self.app.navigate(Route.error(message, title=title))

    def refresh(self) -> None:
        """Reload the data for the current route (manual refresh, FR-13)."""
        route = self.current_route()
        if route.name == RouteName.CHATS and route.server_id is not None:
            self.load_chats(route.server_id)  # type: ignore[attr-defined]
        elif (
            route.name == RouteName.CHAT
            and route.server_id is not None
            and route.chat_uid is not None
        ):
            self.load_chat(route.server_id, route.chat_uid)  # type: ignore[attr-defined]
        elif route.name in (RouteName.SERVERS, RouteName.SERVER):
            # These pages render from local state, so re-navigating reloads them.
            self.app.navigate(route)
        # RouteName.ERROR: nothing to refresh.

    def current_route(self) -> Route:
        """Return the route currently displayed (delegates to the app shell)."""
        return self.app.current_route()

    # --- shared helpers -----------------------------------------------------

    def _persist(self) -> None:
        """Write the current state to the config file.

        A failure here must not crash an otherwise successful action; report it
        as a non-fatal notice/dialog.
        """
        try:
            self.config.save(state_to_config(self.state))
        except ConfigError as exc:
            dialogs.show_error(self.app, "Could not save configuration", str(exc))

    def _server(self, server_id: int) -> ServerConnection:
        """Return the connection or raise :class:`ServerNotFoundError`."""
        return self.state.server_by_id(server_id)

    def _signed_in_server(self, server_id: int) -> ServerConnection | None:
        """Return the connection when it exists **and** is signed in.

        When it is missing, navigate to the error route; when it is signed out,
        navigate back to the server page (the WebUI's "redirect to sign in"
        guard). Returns ``None`` when the caller should stop.
        """
        connection = self.state.find_server(server_id)
        if connection is None:
            self.show_error(f"server not found: {server_id}")
            return None
        if not connection.signed_in:
            self.show_server(server_id)
            return None
        return connection

    def _run(
        self,
        work: Callable[[], Any],
        on_success: Callable[[Any], None],
        *,
        on_error: Callable[[BaseException], None] | None = None,
        on_done: Callable[[], None] | None = None,
    ) -> None:
        """Schedule ``work`` and route failures to the active view.

        The default ``on_error`` shows the error in the current view's banner (or
        a dialog when there is none). Operations override it when a failure should
        re-render a specific view.
        """
        self.tasks.submit(
            work,
            on_success=on_success,
            on_error=on_error if on_error is not None else self._show_api_error,
            on_done=on_done,
        )

    def _show_api_error(self, error: BaseException) -> None:
        """Show an expected failure without leaving the current view."""
        if isinstance(error, APIError):
            message = str(error) or "request failed"
            route = self.current_route()
            connection = (
                self.state.find_server(route.server_id)
                if route.server_id is not None
                else None
            )
            # An operation against a connection that is no longer signed in:
            # send the user back to the connection page to sign in again.
            if connection is not None and not connection.signed_in:
                self.show_server(route.server_id, notice=message)  # type: ignore[arg-type]
                return
            self.app.show_banner_error(message)
            return
        self.show_error(str(error) or error.__class__.__name__)
