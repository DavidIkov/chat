"""One connection's page — parity with ``templates/server.html``.

Signed out: side-by-side Log in / Register forms (FR-4).
Signed in: the uid, an "Open chats" action, "Log out" (FR-5), and the
"Delete account" section with the *also delete my messages* checkbox (FR-15),
shown after the account-deletion confirmation (``?deleted=1`` in the WebUI).

Widget outline::

    <Connection label>            [<- All servers]
    [ Banner: error/notice ]
    -- signed out --                         -- signed in --
    Log in:   Name [__] Password [__] [Go]   Signed in as uid N.
    Register: Name [__] Password [__] [Go]   [Open chats] [Log out]
                                             Delete account: [ ] also delete my
                                             messages            [Delete account]
"""

from __future__ import annotations

from tkinter import ttk

from chat_client.state.session import ServerConnection
from chat_client.ui import dialogs
from chat_client.ui.views.base import BaseView
from chat_client.ui.widgets.forms import LabeledCheckbutton, LabeledEntry


class ServerView(BaseView):
    """Renders login/register or the signed-in summary for one connection."""

    def build(self) -> None:
        # A destructive action gets a visible danger style (the WebUI's
        # ``button.danger``). Named styles are global to the interpreter, so
        # configuring it once here is enough.
        ttk.Style(self).configure("Danger.TButton", foreground="crimson")
        self._container = ttk.Frame(self, padding=16)
        self._container.pack(fill="both", expand=True)

    def on_show(self) -> None:
        """Render the connection resolved from ``route.server_id`` (no backend call).

        If the connection no longer exists, navigate to the error route. A
        ``notice`` carried on the route (post-delete banner) is shown by
        :meth:`render`.
        """
        connection = (
            self.app.state.find_server(self.route.server_id)
            if self.route.server_id is not None
            else None
        )
        if connection is None:
            self.app.controller.show_error("server not found")
            return
        self.render(connection)

    def render(self, connection: ServerConnection) -> None:
        """(Re)build the page for ``connection`` based on its signed-in state."""
        for child in self._container.winfo_children():
            child.destroy()

        header = ttk.Frame(self._container)
        header.pack(fill="x", pady=(0, 8))
        ttk.Label(
            header, text=connection.label, font=("TkDefaultFont", 16, "bold")
        ).pack(side="left")
        ttk.Button(
            header,
            text="\u2190 All servers",
            command=self.app.controller.show_servers,
        ).pack(side="right")

        if self.route.notice:
            self.banner.show_notice(self.route.notice)
        else:
            self.banner.clear()

        if connection.signed_in:
            self._render_signed_in(connection)
        else:
            self._render_signed_out(connection)

    # --- signed out (FR-4) -------------------------------------------------

    def _render_signed_out(self, connection: ServerConnection) -> None:
        """Render the side-by-side Log in / Register forms."""
        columns = ttk.Frame(self._container)
        columns.pack(fill="x")
        columns.columnconfigure(0, weight=1)
        columns.columnconfigure(1, weight=1)

        login = ttk.LabelFrame(columns, text="Log in", padding=8)
        login.grid(row=0, column=0, sticky="nsew", padx=(0, 6))
        self._login_name = LabeledEntry(login, "Name")
        self._login_name.pack(fill="x", pady=2)
        self._login_password = LabeledEntry(login, "Password", show="\u2022")
        self._login_password.pack(fill="x", pady=2)
        ttk.Button(login, text="Log in", command=self._on_login).pack(
            anchor="w", pady=(6, 0)
        )

        register = ttk.LabelFrame(columns, text="Register", padding=8)
        register.grid(row=0, column=1, sticky="nsew", padx=(6, 0))
        self._register_name = LabeledEntry(register, "Name")
        self._register_name.pack(fill="x", pady=2)
        self._register_password = LabeledEntry(register, "Password", show="\u2022")
        self._register_password.pack(fill="x", pady=2)
        ttk.Button(register, text="Register", command=self._on_register).pack(
            anchor="w", pady=(6, 0)
        )

        # Keyboard-first: Return in either password field submits that form and
        # the first entry of the page takes focus on mount.
        self._login_password.bind_return(self._on_login)
        self._register_password.bind_return(self._on_register)
        self._login_name.focus()

    def _on_login(self) -> None:
        if self.route.server_id is None:
            return
        self.app.controller.log_in(
            self.route.server_id,
            self._login_name.get().strip(),
            self._login_password.get(),
        )

    def _on_register(self) -> None:
        if self.route.server_id is None:
            return
        self.app.controller.register(
            self.route.server_id,
            self._register_name.get().strip(),
            self._register_password.get(),
        )

    # --- signed in (FR-5, FR-15) -------------------------------------------

    def _render_signed_in(self, connection: ServerConnection) -> None:
        """Render the signed-in summary, actions and the delete-account form."""
        session = connection.session
        uid = session.uid if session is not None else 0
        ttk.Label(self._container, text=f"Signed in as uid {uid}.").pack(
            anchor="w", pady=(0, 8)
        )

        actions = ttk.Frame(self._container)
        actions.pack(anchor="w", pady=(0, 12))
        ttk.Button(
            actions,
            text="Open chats",
            command=lambda server_id=connection.id: self.app.controller.show_chats(
                server_id
            ),
        ).pack(side="left")
        ttk.Button(
            actions,
            text="Log out",
            command=lambda server_id=connection.id: self.app.controller.log_out(
                server_id
            ),
        ).pack(side="left", padx=(6, 0))

        delete_frame = ttk.LabelFrame(
            self._container, text="Delete account", padding=8
        )
        delete_frame.pack(fill="x", pady=(8, 0))
        ttk.Label(
            delete_frame,
            text=(
                "This permanently deletes your account on this server and "
                "cannot be undone."
            ),
            foreground="#888",
            wraplength=560,
            justify="left",
        ).pack(anchor="w")
        self._delete_messages = LabeledCheckbutton(
            delete_frame, "Also delete my messages from every chat"
        )
        self._delete_messages.pack(anchor="w", pady=(6, 0))
        ttk.Button(
            delete_frame,
            text="Delete account",
            style="Danger.TButton",
            command=self._on_delete,
        ).pack(anchor="w", pady=(8, 0))

    def _on_delete(self) -> None:
        """Confirm and then delete the account on this connection (FR-15)."""
        if self.route.server_id is None:
            return
        connection = self.app.state.find_server(self.route.server_id)
        if connection is None:
            return
        confirmed, dialog_delete_messages = dialogs.confirm_delete_account(
            self, connection.label
        )
        if not confirmed:
            return
        # Either checkbox opting in means the account's messages go too: the
        # inline one mirrors ``server.html``, the dialog's is the confirmation
        # step (it cannot override the inline choice, only add to it).
        delete_messages = self._delete_messages.get() or dialog_delete_messages
        self.app.controller.delete_account(connection.id, delete_messages)
