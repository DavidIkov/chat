"""Landing page — parity with ``templates/index.html``.

Lists every server connection with its name/URL, a "signed in" marker, an
"Open" action and a "Remove" button, plus the "Add an api server" form
(FR-1, FR-2, FR-3).

Widget outline::

    [ Banner: error/notice ]           <- BaseView.show_error/show_notice
    Add server: URL [____] Name [____] [Add]   <- controller.add_server
    -----------------------------------------
    <connection list>
      <label>  <url>  [signed in]  [Open] [Remove]
"""

from __future__ import annotations

from tkinter import ttk

from chat_client.state.session import ServerConnection
from chat_client.ui.views.base import BaseView
from chat_client.ui.widgets.forms import LabeledEntry


class ServersView(BaseView):
    """Renders the connection list and the add-server form."""

    def build(self) -> None:
        container = ttk.Frame(self, padding=16)
        container.pack(fill="both", expand=True)

        ttk.Label(
            container, text="Servers", font=("TkDefaultFont", 16, "bold")
        ).pack(anchor="w", pady=(0, 8))

        add_frame = ttk.LabelFrame(container, text="Add an api server", padding=8)
        add_frame.pack(fill="x", pady=(0, 12))

        self._url = LabeledEntry(
            add_frame, "URL", placeholder="http://localhost:8080"
        )
        self._url.pack(fill="x", pady=2)
        self._url.bind_return(self._on_add)

        self._name = LabeledEntry(add_frame, "Name", placeholder="optional")
        self._name.pack(fill="x", pady=2)
        self._name.bind_return(self._on_add)

        self._add_button = ttk.Button(
            add_frame, text="Add server", command=self._on_add
        )
        self._add_button.pack(anchor="w", pady=(6, 0))

        self._list = ttk.Frame(container)
        self._list.pack(fill="both", expand=True)

    def on_show(self) -> None:
        """Render ``app.state.list_servers()`` (no backend call)."""
        if self.route.notice:
            self.banner.show_notice(self.route.notice)
        else:
            self.banner.clear()
        self.render_servers(self.app.state.list_servers())
        # First entry of the page takes focus (keyboard-first, like the WebUI's
        # autofocused form field).
        self._url.focus()

    def render_servers(self, servers: list[ServerConnection]) -> None:
        """(Re)build the connection list from ``servers``.

        Shows an empty-state hint when the list is empty, matching the WebUI's
        "No servers yet. Add one above.".
        """
        for child in self._list.winfo_children():
            child.destroy()

        if not servers:
            ttk.Label(
                self._list,
                text="No servers yet. Add one above.",
                foreground="#888",
            ).pack(anchor="w")
            return

        for connection in servers:
            self._render_row(connection)

    # --- internals ---------------------------------------------------------

    def _on_add(self) -> None:
        """Forward the add-server form to the controller."""
        self.app.controller.add_server(self._url.get(), self._name.get())

    def _render_row(self, connection: ServerConnection) -> None:
        """Render one connection: label, url, signed-in marker and actions."""
        row = ttk.Frame(self._list)
        row.pack(fill="x", pady=3)
        row.columnconfigure(0, weight=1)

        details = ttk.Frame(row)
        details.grid(row=0, column=0, sticky="w")
        ttk.Label(
            details, text=connection.label, font=("TkDefaultFont", 11, "bold")
        ).pack(side="left")
        ttk.Label(details, text=connection.url, foreground="#888").pack(
            side="left", padx=(8, 0)
        )
        if connection.signed_in:
            ttk.Label(details, text="signed in", foreground="green").pack(
                side="left", padx=(8, 0)
            )

        actions = ttk.Frame(row)
        actions.grid(row=0, column=1, sticky="e")
        ttk.Button(
            actions,
            text="Open",
            command=lambda server_id=connection.id: self.app.controller.show_server(
                server_id
            ),
        ).pack(side="left")
        ttk.Button(
            actions,
            text="Remove",
            command=lambda server_id=connection.id: self.app.controller.remove_server(
                server_id
            ),
        ).pack(side="left", padx=(6, 0))
