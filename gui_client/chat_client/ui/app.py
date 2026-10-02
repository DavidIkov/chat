"""The application shell: Tk root, router, menu and status bar.

This is the client's composition root and the analogue of
``webui_server``'s ``http.ServeMux`` + ``render.Render``/``render.Redirect``:
it owns the window chrome and swaps the visible :class:`BaseView` when the
controller navigates.

Construction order (see :mod:`chat_client.__main__`):

1. ``state``/``api``/``config`` are built and the config is loaded;
2. ``App(...)`` is created — it builds the :class:`TaskRunner` (which needs a
   widget for ``after``) and the :class:`Controller`;
3. the controller immediately navigates to the landing page.

``App`` is the only object that knows about every view class.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk

from chat_client.apiclient.client import ApiClient
from chat_client.config.store import ConfigStore
from chat_client.constants import (
    APP_NAME,
    DEFAULT_WINDOW_SIZE,
    MIN_WINDOW_SIZE,
    REFRESH_HINT,
    WINDOW_TITLE,
)
from chat_client.controller.controller import Controller
from chat_client.state.session import AppState
from chat_client.tasks.runner import TaskRunner
from chat_client.ui import dialogs
from chat_client.ui.route import Route, RouteName
from chat_client.ui.views.chat import ChatView
from chat_client.ui.views.chats import ChatsView
from chat_client.ui.views.error import ErrorView
from chat_client.ui.views.server import ServerView
from chat_client.ui.views.servers import ServersView


class App(tk.Tk):
    """Main window and view router."""

    #: Every route name maps to the view class that renders it.
    VIEWS: dict[str, type] = {
        RouteName.SERVERS: ServersView,
        RouteName.SERVER: ServerView,
        RouteName.CHATS: ChatsView,
        RouteName.CHAT: ChatView,
        RouteName.ERROR: ErrorView,
    }

    def __init__(self, state: AppState, api: ApiClient, config: ConfigStore) -> None:
        super().__init__()
        self.title(WINDOW_TITLE)
        self.geometry(f"{DEFAULT_WINDOW_SIZE[0]}x{DEFAULT_WINDOW_SIZE[1]}")
        self.minsize(*MIN_WINDOW_SIZE)

        self.state = state
        self.api = api
        self.config = config

        self._build_menu()

        # Status bar is packed before the content so it always stays at the
        # bottom of the window.
        self._status_var = tk.StringVar(value=REFRESH_HINT)
        status_bar = ttk.Frame(self)
        status_bar.pack(side="bottom", fill="x")
        ttk.Label(status_bar, textvariable=self._status_var, anchor="w").pack(
            side="left", fill="x", expand=True, padx=6, pady=2
        )

        self._content = ttk.Frame(self)
        self._content.pack(fill="both", expand=True)

        self.tasks = TaskRunner(self)
        self.controller = Controller(self, state, api, self.tasks, config)

        self._view = None
        self._route: Route | None = None
        self.protocol("WM_DELETE_WINDOW", self.on_close)
        self.bind("<F5>", lambda _event: self.controller.refresh())

        self.controller.show_servers()

    # --- chrome -------------------------------------------------------------

    def _build_menu(self) -> None:
        menubar = tk.Menu(self, tearoff=0)

        file_menu = tk.Menu(menubar, tearoff=0)
        file_menu.add_command(
            label="Add server", command=lambda: self.controller.show_servers()
        )
        file_menu.add_command(
            label="Refresh", accelerator="F5", command=lambda: self.controller.refresh()
        )
        file_menu.add_separator()
        file_menu.add_command(label="Quit", command=self.on_close)
        menubar.add_cascade(label="File", menu=file_menu)

        help_menu = tk.Menu(menubar, tearoff=0)
        help_menu.add_command(label="About", command=self._show_about)
        menubar.add_cascade(label="Help", menu=help_menu)

        self.configure(menu=menubar)
        self._menubar = menubar

    def _show_about(self) -> None:
        dialogs.show_info(
            self,
            f"About {APP_NAME}",
            f"{APP_NAME} — desktop client for the Chat api_server.\n\n"
            f"{REFRESH_HINT}",
        )

    def _update_title(self, route: Route) -> None:
        label = ""
        if route.server_id is not None:
            connection = self.state.find_server(route.server_id)
            if connection is not None:
                label = connection.label
        self.title(f"{WINDOW_TITLE} — {label}" if label else WINDOW_TITLE)

    # --- routing ------------------------------------------------------------

    def navigate(self, route: Route) -> None:
        """Destroy the current view and mount the one selected by ``route``.

        Instantiates the view class registered for ``route.name``, packs it,
        stores it as the current view and calls its ``on_show()`` hook. Failures
        raised while rendering are caught and turned into an error route so a
        single bad view cannot take down the window.
        """
        if self._view is not None:
            self._view.destroy()
            self._view = None

        view_cls = self.VIEWS.get(route.name)
        if view_cls is None:
            route = Route.error(f"unknown route: {route.name!r}")
            view_cls = ErrorView

        try:
            view = view_cls(self._content, self, route)
            view.pack(fill="both", expand=True)
        except Exception as exc:  # noqa: BLE001 - any render failure -> error page
            if route.name == RouteName.ERROR:
                dialogs.show_error(self, route.title, str(exc) or repr(exc))
                return
            self.navigate(Route.error(str(exc) or exc.__class__.__name__))
            return

        self._view = view
        self._route = route
        self._update_title(route)
        self.set_status(REFRESH_HINT)

        try:
            view.on_show()
        except Exception as exc:  # noqa: BLE001 - a load failure -> error page
            if route.name == RouteName.ERROR:
                view.show_error(str(exc) or repr(exc))
                return
            view.destroy()
            self._view = None
            self.navigate(Route.error(str(exc) or exc.__class__.__name__))

    def current_route(self) -> Route:
        """Return the route currently displayed."""
        if self._route is not None:
            return self._route
        return Route.servers()

    # --- view protocol ------------------------------------------------------

    def show_banner_error(self, message: str) -> None:
        """Show ``message`` in the current view's error area (fallback: dialog)."""
        if self._view is not None:
            self._view.show_error(message)
        else:
            dialogs.show_error(self, "Error", message)

    def show_banner_notice(self, message: str) -> None:
        """Show ``message`` in the current view's notice area."""
        if self._view is not None:
            self._view.show_notice(message)
        else:
            dialogs.show_info(self, APP_NAME, message)

    def set_status(self, text: str) -> None:
        """Set the status-bar text (e.g. the active connection's label)."""
        self._status_var.set(text)

    # --- lifecycle ----------------------------------------------------------

    def on_close(self) -> None:
        """Shut the task runner down and destroy the window."""
        try:
            self.tasks.shutdown()
        finally:
            self.destroy()
