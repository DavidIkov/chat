"""One chat — parity with ``templates/chat.html``.

The richest view: chat name, the join-link token banner (shown once), the message
list with resolved author names, the send form (FR-10), the member list
(FR-9), the join-link form with opt-in lifetime/max-uses (FR-11) and the leave
action (FR-12).

Widget outline::

    <Chat name>                    [<- Chats] [Refresh]
    [ Banner: error/notice ]
    [ Join link token: <code>  ]      <- shown once (see PendingJoinLink)
    Messages:
        <author>  <text>
    Send a message: [__________________________] [Send]
    Members: <name> / uid N
    Share:
        [ ] Expire the link after a set time   Lifetime [1 day v]
        [ ] Limit the number of uses           Max uses [10]
        [Create join link]
    [Leave chat]

The view never talks to the network: :meth:`on_show` asks the controller to load
the chat and the controller calls :meth:`render_chat` with fully resolved data
(author/member names already turned from uids where possible).
"""

from __future__ import annotations

from tkinter import ttk

from chat_client.constants import (
    JOIN_LINK_DEFAULT_MAX_USES,
    JOIN_LINK_LIFETIME_OPTIONS,
)
from chat_client.models.chat import Chat, Member, Message
from chat_client.ui import dialogs
from chat_client.ui.views.base import BaseView
from chat_client.ui.widgets.forms import (
    LabeledCheckbutton,
    LabeledCombobox,
    LabeledEntry,
    LabeledSpinbox,
)
from chat_client.ui.widgets.lists import MemberList, MessageList


class ChatView(BaseView):
    """Renders one chat and hosts its actions."""

    def build(self) -> None:
        # The destructive action reuses the shared danger style (ServerView also
        # configures it; configuring a named style twice is harmless).
        ttk.Style(self).configure("Danger.TButton", foreground="crimson")

        self._container = ttk.Frame(self, padding=16)
        self._container.pack(fill="both", expand=True)

        header = ttk.Frame(self._container)
        header.pack(fill="x", pady=(0, 8))
        self._title = ttk.Label(
            header, text="Chat", font=("TkDefaultFont", 16, "bold")
        )
        self._title.pack(side="left")
        ttk.Button(header, text="Refresh", command=self._on_refresh).pack(side="right")
        ttk.Button(header, text="\u2190 Chats", command=self._on_back).pack(
            side="right", padx=(0, 6)
        )

        # Join-link strip: created now, packed on demand before the Messages
        # heading so it can appear/disappear without reordering the page.
        self._join_link_frame = ttk.Frame(self._container)
        ttk.Label(
            self._join_link_frame, text="Join link token: ", foreground="green"
        ).pack(side="left")
        self._join_link_code = ttk.Label(
            self._join_link_frame,
            text="",
            foreground="green",
            font=("TkFixedFont",),
        )
        self._join_link_code.pack(side="left")

        self._messages_heading = ttk.Label(
            self._container, text="Messages", font=("TkDefaultFont", 12, "bold")
        )
        self._messages_heading.pack(anchor="w", pady=(8, 4))
        self._messages = MessageList(self._container, height=220)
        self._messages.pack(fill="both", expand=True)

        send_frame = ttk.LabelFrame(self._container, text="Send a message", padding=8)
        send_frame.pack(fill="x", pady=(8, 0))
        self._text = LabeledEntry(send_frame, "Message", width=48)
        self._text.pack(fill="x", pady=2)
        self._text.bind_return(self._on_send)
        ttk.Button(send_frame, text="Send", command=self._on_send).pack(
            anchor="w", pady=(6, 0)
        )

        ttk.Label(
            self._container, text="Members", font=("TkDefaultFont", 12, "bold")
        ).pack(anchor="w", pady=(8, 4))
        self._members = MemberList(self._container)
        self._members.pack(fill="x")

        share_frame = ttk.LabelFrame(self._container, text="Share", padding=8)
        share_frame.pack(fill="x", pady=(8, 0))
        ttk.Label(
            share_frame,
            text=(
                "By default the link never expires and can be used any number of "
                "times. Tick an option to limit it."
            ),
            foreground="#888",
            wraplength=560,
            justify="left",
        ).pack(anchor="w")

        self._limit_lifetime = LabeledCheckbutton(
            share_frame, "Expire the link after a set time"
        )
        self._limit_lifetime.pack(anchor="w", pady=(6, 0))
        self._lifetime = LabeledCombobox(
            share_frame, "Lifetime", JOIN_LINK_LIFETIME_OPTIONS
        )
        self._lifetime.pack(anchor="w", padx=(20, 0), pady=2)

        self._limit_uses = LabeledCheckbutton(
            share_frame, "Limit the number of uses"
        )
        self._limit_uses.pack(anchor="w", pady=(6, 0))
        self._max_uses = LabeledSpinbox(
            share_frame, "Max uses", from_=1, value=JOIN_LINK_DEFAULT_MAX_USES
        )
        self._max_uses.pack(anchor="w", padx=(20, 0), pady=2)

        # Keep each dependent field disabled while its checkbox is off.
        self._limit_lifetime.var.trace_add("write", lambda *_: self._sync_share_fields())
        self._limit_uses.var.trace_add("write", lambda *_: self._sync_share_fields())
        self._sync_share_fields()

        ttk.Button(
            share_frame, text="Create join link", command=self._on_create_join_link
        ).pack(anchor="w", pady=(8, 0))

        ttk.Button(
            self._container,
            text="Leave chat",
            style="Danger.TButton",
            command=self._on_leave,
        ).pack(anchor="w", pady=(8, 0))

    def on_show(self) -> None:
        """Ask the controller to load the chat (``load_chat``); performs no I/O."""
        server_id = self.route.server_id
        chat_uid = self.route.chat_uid
        if server_id is None or chat_uid is None:
            self.app.controller.show_error("chat not found")
            return
        self._show_loading()
        self.app.controller.load_chat(server_id, chat_uid)

    def render_chat(
        self,
        chat: Chat,
        messages: list["ChatMessageItem"],
        members: list["ChatMemberItem"],
        join_link: str,
    ) -> None:
        """Render the chat, its messages/members and (optionally) a join token."""
        self._title.configure(text=chat.name)
        self._messages.set_messages(
            [
                (item.user_name or f"uid {item.message.user_uid}", item.message.text)
                for item in messages
            ]
        )
        self._members.set_members(
            [member.name or f"uid {member.uid}" for member in members]
        )
        # A reload is the client's Post/Redirect/Get: the form starts empty again.
        self._text.set("")
        if join_link:
            self.set_join_link(join_link)
        else:
            self._hide_join_link()

    def set_join_link(self, token: str) -> None:
        """Show a freshly created join token near the top (once)."""
        self._join_link_code.configure(text=token)
        if not self._join_link_frame.winfo_manager():
            self._join_link_frame.pack(
                fill="x", pady=(0, 8), before=self._messages_heading
            )

    # --- internals ---------------------------------------------------------

    def _hide_join_link(self) -> None:
        self._join_link_code.configure(text="")
        self._join_link_frame.pack_forget()

    def _show_loading(self) -> None:
        self.banner.clear()
        self._messages.show_empty("Loading\u2026")
        self._text.focus()

    def _sync_share_fields(self) -> None:
        self._lifetime.set_enabled(self._limit_lifetime.get())
        self._max_uses.set_enabled(self._limit_uses.get())

    def _on_back(self) -> None:
        if self.route.server_id is not None:
            self.app.controller.show_chats(self.route.server_id)

    def _on_refresh(self) -> None:
        if self.route.server_id is not None and self.route.chat_uid is not None:
            self._show_loading()
            self.app.controller.load_chat(self.route.server_id, self.route.chat_uid)

    def _on_send(self) -> None:
        if self.route.server_id is not None and self.route.chat_uid is not None:
            self.app.controller.send_message(
                self.route.server_id, self.route.chat_uid, self._text.get()
            )

    def _on_create_join_link(self) -> None:
        server_id = self.route.server_id
        chat_uid = self.route.chat_uid
        if server_id is None or chat_uid is None:
            return

        lifetime = 0
        if self._limit_lifetime.get():
            value = self._lifetime.get()
            lifetime = int(value) if isinstance(value, int) else 0
            if lifetime <= 0:
                self.banner.show_error("Lifetime must be a positive number of seconds.")
                return

        max_uses = 0
        if self._limit_uses.get():
            max_uses = self._max_uses.get()
            if max_uses <= 0:
                self.banner.show_error("Max uses must be a positive number.")
                return

        self.app.controller.create_join_link(server_id, chat_uid, lifetime, max_uses)

    def _on_leave(self) -> None:
        server_id = self.route.server_id
        chat_uid = self.route.chat_uid
        if server_id is None or chat_uid is None:
            return
        if not dialogs.confirm(
            self,
            "Leave chat",
            "Leave this chat? You can only rejoin with a new invite link.",
        ):
            return
        self.app.controller.leave_chat(server_id, chat_uid)


class ChatMessageItem:
    """One rendered message: the message plus its resolved author name."""

    def __init__(self, message: Message, user_name: str = "") -> None:
        self.message = message
        self.user_name = user_name


class ChatMemberItem:
    """One rendered member: uid plus its resolved name."""

    def __init__(self, uid: int, name: str = "") -> None:
        self.uid = uid
        self.name = name
