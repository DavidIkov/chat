"""Error page — parity with ``templates/error.html``.

Rendered for unexpected failures (missing connection, malformed ids, backend
errors during a load that cannot be shown inline). Shows the title, the message
and a "Back to servers" action.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk

from chat_client.ui.route import Route
from chat_client.ui.views.base import BaseView


class ErrorView(BaseView):
    """Renders ``route.title`` and ``route.message``."""

    def __init__(self, master: tk.Misc, app: "App", route: Route) -> None:  # noqa: F821
        super().__init__(master, app, route)

    def build(self) -> None:
        container = ttk.Frame(self, padding=16)
        container.pack(fill="both", expand=True)

        self._title = ttk.Label(
            container, text="", font=("TkDefaultFont", 14, "bold")
        )
        self._title.pack(anchor="w", pady=(0, 8))

        self._message = ttk.Label(
            container,
            text="",
            foreground="crimson",
            wraplength=640,
            justify="left",
        )
        self._message.pack(anchor="w")

        ttk.Button(
            container,
            text="Back to servers",
            command=lambda: self.app.controller.show_servers(),
        ).pack(anchor="w", pady=(16, 0))

    def on_show(self) -> None:
        """Populate the view from ``self.route.message``/``title``."""
        self._title.configure(text=self.route.title or "Error")
        self._message.configure(text=self.route.message)
