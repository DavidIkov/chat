"""An inline error/notice strip used at the top of a page.

Clients the WebUI's ``<p class="error">`` / ``<p class="notice">``: the controller
re-renders the *current* page with a message instead of navigating to a bare
error page, so the user can correct their input.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk


class Banner(ttk.Frame):
    """A one-line message area with error and notice variants.

    ``show_error``/``show_notice`` display the message; ``clear`` hides it.
    Colours follow the WebUI's crimson/green convention from ``style.css``.
    """

    _ERROR_COLOR = "crimson"
    _NOTICE_COLOR = "green"

    def __init__(self, master: tk.Misc) -> None:
        super().__init__(master)
        self._message = ""
        self._label = ttk.Label(self, anchor="w", justify="left")
        self._label_shown = False
        # Keep the message inside the frame as the window is resized.
        self.bind("<Configure>", self._on_configure)

    def _on_configure(self, event: "tk.Event") -> None:
        self._label.configure(wraplength=max(120, int(event.width) - 16))

    def _is_packed(self) -> bool:
        try:
            self.pack_info()
        except tk.TclError:
            return False
        return True

    def _mount(self) -> None:
        """Pack the strip at the top of the parent, on demand."""
        if self._is_packed():
            return
        siblings = [widget for widget in self.master.pack_slaves() if widget is not self]
        if siblings:
            self.pack(fill="x", before=siblings[0])
        else:
            self.pack(fill="x")

    def _show(self, message: str, color: str) -> None:
        self._mount()
        if not self._label_shown:
            self._label.pack(fill="x", padx=6, pady=3)
            self._label_shown = True
        self._message = message
        self._label.configure(text=message, foreground=color)

    def show_error(self, message: str) -> None:
        self._show(message, self._ERROR_COLOR)

    def show_notice(self, message: str) -> None:
        self._show(message, self._NOTICE_COLOR)

    def clear(self) -> None:
        self._message = ""
        if self._label_shown:
            self._label.pack_forget()
            self._label_shown = False
        self.pack_forget()

    @property
    def message(self) -> str:
        """The currently displayed text (empty when hidden)."""
        return self._message
