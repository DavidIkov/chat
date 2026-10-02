"""A vertically scrollable container.

Tkinter has no scrollable frame; this composes a ``Canvas`` + inner frame +
``ttk.Scrollbar`` and wires mouse-wheel scrolling. Used for the message list,
where content easily exceeds the window height.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk


class ScrollableFrame(ttk.Frame):
    """A frame whose ``body`` can grow taller than the visible area.

    Put children into ``self.body``; the frame scrolls when they exceed the
    viewport. ``refresh()`` recalculates the scroll region (call it after adding
    or removing ``body`` children).
    """

    def __init__(self, master: tk.Misc, *, height: int = 240) -> None:
        """Create the frame; children go into ``self.body``."""
        super().__init__(master)

        self._canvas = tk.Canvas(
            self, height=height, highlightthickness=0, borderwidth=0
        )
        self._scrollbar = ttk.Scrollbar(
            self, orient="vertical", command=self._canvas.yview
        )
        self._canvas.configure(yscrollcommand=self._scrollbar.set)

        self._scrollbar.pack(side="right", fill="y")
        self._canvas.pack(side="left", fill="both", expand=True)

        self.body = ttk.Frame(self._canvas)
        self._window = self._canvas.create_window((0, 0), window=self.body, anchor="nw")

        self.body.bind("<Configure>", self._on_body_configure)
        self._canvas.bind("<Configure>", self._on_canvas_configure)

        # ``add="+"`` lets several ScrollableFrames coexist: every instance gets
        # the event and only the one under the pointer reacts (see below).
        for sequence in ("<MouseWheel>", "<Button-4>", "<Button-5>"):
            self._canvas.bind_all(sequence, self._on_mousewheel, add="+")

        self.refresh()

    # --- geometry ----------------------------------------------------------

    def _on_body_configure(self, _event: "tk.Event") -> None:
        self._canvas.configure(scrollregion=self._canvas.bbox("all"))

    def _on_canvas_configure(self, event: "tk.Event") -> None:
        # Make the inner body fill the canvas width so text wraps to the view.
        self._canvas.itemconfigure(self._window, width=int(event.width))

    def refresh(self) -> None:
        """Recompute the scroll region after the body's content changes."""
        try:
            self.body.update_idletasks()
            self._canvas.configure(scrollregion=self._canvas.bbox("all"))
        except tk.TclError:  # pragma: no cover - widget already destroyed
            pass

    # --- mouse wheel -------------------------------------------------------

    def _pointer_inside(self, event: "tk.Event") -> bool:
        try:
            left = self.winfo_rootx()
            top = self.winfo_rooty()
            right = left + self.winfo_width()
            bottom = top + self.winfo_height()
        except tk.TclError:  # pragma: no cover - widget already destroyed
            return False
        return left <= event.x_root < right and top <= event.y_root < bottom

    def _on_mousewheel(self, event: "tk.Event") -> None:
        if not self._pointer_inside(event):
            return
        num = getattr(event, "num", None)
        delta = getattr(event, "delta", 0)
        if num == 4 or delta > 0:
            direction = -1
        elif num == 5 or delta < 0:
            direction = 1
        else:
            return
        self._canvas.yview_scroll(direction, "units")
