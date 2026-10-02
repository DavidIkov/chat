"""Read-only list widgets for messages and members.

Both accept already-resolved display data from the chat view (the controller has
already turned uids into names), so they contain no HTTP or state knowledge.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk

from chat_client.ui.widgets.scrollable import ScrollableFrame


class MessageList(ttk.Frame):
    """Scrollable list of ``(author, text)`` rows.

    ``set_messages`` clears and rebuilds the rows. A missing author name should
    fall back to ``uid N``, mirroring ``templates/chat.html``.
    """

    def __init__(self, master: tk.Misc, *, height: int = 240) -> None:
        super().__init__(master)
        self._scroll = ScrollableFrame(self, height=height)
        self._scroll.pack(fill="both", expand=True)

    def _clear(self) -> None:
        for child in self._scroll.body.winfo_children():
            child.destroy()

    def set_messages(self, items: list[tuple[str, str]]) -> None:
        """Render ``items`` as ``(author_label, text)`` pairs, oldest first."""
        self._clear()
        if not items:
            self.show_empty()
            return
        for author, text in items:
            row = ttk.Frame(self._scroll.body)
            row.pack(fill="x", pady=2)
            row.columnconfigure(1, weight=1)
            ttk.Label(
                row, text=author, font=("TkDefaultFont", 9, "bold")
            ).grid(row=0, column=0, sticky="nw", padx=(0, 6))
            text_label = ttk.Label(row, text=text, justify="left", wraplength=620)
            text_label.grid(row=0, column=1, sticky="nw")
            row.bind(
                "<Configure>",
                lambda event, label=text_label: label.configure(
                    wraplength=max(120, int(event.width) - 120)
                ),
            )
        self._scroll.refresh()

    def show_empty(self, text: str = "No messages yet.") -> None:
        self._clear()
        ttk.Label(self._scroll.body, text=text, foreground="#888").pack(
            anchor="w", pady=4
        )
        self._scroll.refresh()


class MemberList(ttk.Frame):
    """Simple list of member display names."""

    def __init__(self, master: tk.Misc) -> None:
        super().__init__(master)
        self._body = ttk.Frame(self)
        self._body.pack(fill="both", expand=True)

    def set_members(self, names: list[str]) -> None:
        """Render member labels, falling back to ``uid N`` when a name is unknown."""
        for child in self._body.winfo_children():
            child.destroy()
        if not names:
            ttk.Label(self._body, text="No members.", foreground="#888").pack(
                anchor="w"
            )
            return
        for name in names:
            ttk.Label(self._body, text=name).pack(anchor="w")
