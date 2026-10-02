"""Chat list — parity with ``templates/chats.html``.

Shows every chat the signed-in user belongs to (FR-6), the "Create chat" form
(FR-7) and the "Join chat by token" form (FR-8). Opening a chat navigates to the
chat view.

Widget outline::

    Chats — <Connection label>          [<- Server] [Refresh]
    [ Banner: error/notice ]
    Create chat: Name [______] [Create]
    Join chat:   Token [______] [Join]
    -----------------------------------------
    <chat list>   each row links to Route.chat(server_id, chat_uid)

The view performs no I/O: :meth:`on_show` resolves the connection only for the
header label and then asks the controller to load the list.
"""

from __future__ import annotations

from tkinter import ttk

from chat_client.models.chat import Chat
from chat_client.ui.views.base import BaseView
from chat_client.ui.widgets.forms import LabeledEntry
from chat_client.ui.widgets.scrollable import ScrollableFrame


class ChatsView(BaseView):
    """Renders the chat list and the create/join forms."""

    def build(self) -> None:
        self._container = ttk.Frame(self, padding=16)
        self._container.pack(fill="both", expand=True)

        header = ttk.Frame(self._container)
        header.pack(fill="x", pady=(0, 8))
        self._title = ttk.Label(
            header, text="Chats", font=("TkDefaultFont", 16, "bold")
        )
        self._title.pack(side="left")
        ttk.Button(header, text="Refresh", command=self._on_refresh).pack(
            side="right"
        )
        ttk.Button(header, text="\u2190 Server", command=self._on_back).pack(
            side="right", padx=(0, 6)
        )

        create_frame = ttk.LabelFrame(self._container, text="Create chat", padding=8)
        create_frame.pack(fill="x", pady=(0, 8))
        self._name = LabeledEntry(create_frame, "Name")
        self._name.pack(fill="x", pady=2)
        self._name.bind_return(self._on_create)
        ttk.Button(create_frame, text="Create", command=self._on_create).pack(
            anchor="w", pady=(6, 0)
        )

        join_frame = ttk.LabelFrame(self._container, text="Join chat", padding=8)
        join_frame.pack(fill="x", pady=(0, 12))
        self._token = LabeledEntry(join_frame, "Token")
        self._token.pack(fill="x", pady=2)
        self._token.bind_return(self._on_join)
        ttk.Button(join_frame, text="Join", command=self._on_join).pack(
            anchor="w", pady=(6, 0)
        )

        self._scroll = ScrollableFrame(self._container, height=200)
        self._scroll.pack(fill="both", expand=True)
        self._list = self._scroll.body

    def on_show(self) -> None:
        """Label the header and ask the controller to load the chat list.

        The connection is resolved from ``app.state`` purely for the display
        label; the actual fetch is the controller's ``load_chats`` (which repeats
        the sign-in guard and navigates to the server page when signed out).
        """
        server_id = self.route.server_id
        if server_id is None:
            self.app.controller.show_error("server not found")
            return
        connection = self.app.state.find_server(server_id)
        if connection is None:
            self.app.controller.show_error("server not found")
            return
        self._title.configure(text=f"Chats \u2014 {connection.label}")
        self._show_loading()
        self._name.focus()
        self.app.controller.load_chats(server_id)

    def render_chats(self, chats: list[Chat]) -> None:
        """(Re)build the list, with an empty-state message when there are none.

        Any previously rendered rows are cleared first so a refresh in place
        does not stack duplicate entries.
        """
        self._clear_list()
        if not chats:
            self.show_empty()
            return
        for chat in chats:
            ttk.Button(
                self._list,
                text=chat.name,
                command=lambda uid=chat.chat_uid: self._open_chat(uid),
            ).pack(fill="x", pady=2)
        self._scroll.refresh()

    def show_empty(self) -> None:
        """Show the "No chats yet." empty state."""
        self._clear_list()
        ttk.Label(self._list, text="No chats yet.", foreground="#888").pack(
            anchor="w", pady=4
        )
        self._scroll.refresh()

    # --- internals ---------------------------------------------------------

    def _clear_list(self) -> None:
        for child in self._list.winfo_children():
            child.destroy()

    def _show_loading(self) -> None:
        """Show a "Loading…" placeholder until the list (or an error) arrives."""
        self.banner.clear()
        self._clear_list()
        ttk.Label(self._list, text="Loading\u2026", foreground="#888").pack(
            anchor="w", pady=4
        )
        self._scroll.refresh()

    def _open_chat(self, chat_uid: int) -> None:
        if self.route.server_id is not None:
            self.app.controller.show_chat(self.route.server_id, chat_uid)

    def _on_back(self) -> None:
        if self.route.server_id is not None:
            self.app.controller.show_server(self.route.server_id)

    def _on_refresh(self) -> None:
        if self.route.server_id is None:
            return
        self._show_loading()
        self.app.controller.load_chats(self.route.server_id)

    def _on_create(self) -> None:
        if self.route.server_id is not None:
            self.app.controller.create_chat(self.route.server_id, self._name.get())

    def _on_join(self) -> None:
        if self.route.server_id is not None:
            self.app.controller.join_chat(self.route.server_id, self._token.get())
