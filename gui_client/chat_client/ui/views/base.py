"""Shared base class for every page view.

Provides the small contract the router relies on:

* ``on_show()`` — called by :meth:`chat_client.ui.app.App.navigate` right after
  the view is mounted; the natural place to kick off a data load via the
  controller (the client's "handler runs on GET").
* ``show_error()`` / ``show_notice()`` — used by the controller to report the
  outcome of an action without re-navigating (the client's "re-render the page
  with an error message").
* ``set_loading()`` — toggles a busy indicator so slow requests are visible
  without freezing.
* ``build()`` — subclass hook that constructs the view's widgets; ``__init__``
  calls it once, after creating the shared ``self.banner``.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk

from chat_client.ui.route import Route
from chat_client.ui.widgets.banner import Banner


class BaseView(ttk.Frame):
    """Base class for page frames.

    :param master: parent widget (the app's content frame).
    :param app: the :class:`~chat_client.ui.app.App` shell, used for
        ``app.controller``/``app.navigate``.
    :param route: the route this view was created for.
    """

    def __init__(self, master: tk.Misc, app: "App", route: Route) -> None:  # noqa: F821
        super().__init__(master)
        self.app = app
        self.route = route
        self.banner = Banner(self)
        self.banner.pack(fill="x")
        self._busy_label: ttk.Label | None = None
        self._loading = False
        self.build()

    def build(self) -> None:
        """Construct the view's widgets. Called once by ``__init__``.

        Subclasses override this instead of ``__init__`` so the base can create
        ``self.banner`` and store ``app``/``route`` first. The default is a no-op
        for views that have nothing to build.
        """

    def on_show(self) -> None:
        """Hook invoked once after mounting. Default: do nothing."""

    def show_error(self, message: str) -> None:
        """Display an error message for a failed action (no navigation)."""
        self.banner.show_error(message)

    def show_notice(self, message: str) -> None:
        """Display a success/notice message (no navigation)."""
        self.banner.show_notice(message)

    def set_loading(self, loading: bool) -> None:
        """Toggle the view's busy state (disable forms, show a spinner/label)."""
        self._loading = loading
        if loading:
            if self._busy_label is None:
                self._busy_label = ttk.Label(
                    self, text="Working…", foreground="#888"
                )
            if not self._busy_label.winfo_manager():
                self._busy_label.pack(anchor="w", padx=6, pady=2)
        elif self._busy_label is not None:
            self._busy_label.pack_forget()
