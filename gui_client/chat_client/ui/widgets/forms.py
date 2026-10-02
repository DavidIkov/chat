"""Small labelled form-field widgets.

The WebUI renders ``<label>Text <input></label>`` pairs; these wrap the same idea
for ``ttk`` so page views stay declarative. All of them expose ``get()``/``set()``
(and ``var`` for checkbuttons) so views never touch tkinter variables directly.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import ttk
from typing import Callable, Sequence


class LabeledEntry(ttk.Frame):
    """A ``ttk.Label`` next to a ``ttk.Entry``.

    ``show`` is passed through to the entry so the same class serves password
    fields (``show="•"``).
    """

    def __init__(
        self,
        master: tk.Misc,
        label: str,
        *,
        show: str | None = None,
        width: int = 28,
        placeholder: str = "",
    ) -> None:
        super().__init__(master)
        self._placeholder = placeholder
        self._var = tk.StringVar()

        ttk.Label(self, text=label).pack(side="left", padx=(0, 4))
        self._entry = ttk.Entry(self, textvariable=self._var, width=width)
        if show is not None:
            self._entry.configure(show=show)
        self._entry.pack(side="left", fill="x", expand=True)

        if placeholder:
            self._var.set(placeholder)
            self._entry.configure(foreground="#888")
            self._entry.bind("<FocusIn>", self._on_focus_in)
            self._entry.bind("<FocusOut>", self._on_focus_out)

    def _on_focus_in(self, _event: "tk.Event") -> None:
        if self._var.get() == self._placeholder:
            self._var.set("")
            self._entry.configure(foreground="")

    def _on_focus_out(self, _event: "tk.Event") -> None:
        if not self._var.get():
            self._var.set(self._placeholder)
            self._entry.configure(foreground="#888")

    def get(self) -> str:
        value = self._var.get()
        return "" if value == self._placeholder else value

    def set(self, value: str) -> None:
        self._var.set(value)
        self._entry.configure(foreground="")

    def focus(self) -> None:
        self._entry.focus_set()

    def bind_return(self, callback: "Callable[[], None]") -> None:
        """Invoke ``callback`` when Return is pressed in the entry.

        Lets a page submit its form from the keyboard (the WebUI's implicit form
        submit) without the view reaching into the entry widget.
        """
        self._entry.bind("<Return>", lambda _event: callback())

    def set_enabled(self, enabled: bool) -> None:
        """Enable or disable the entry (used for dependent fields)."""
        self._entry.configure(state="normal" if enabled else "disabled")


class LabeledCheckbutton(ttk.Frame):
    """A labelled ``ttk.Checkbutton`` bound to a ``tk.BooleanVar``."""

    def __init__(self, master: tk.Misc, label: str, *, value: bool = False) -> None:
        super().__init__(master)
        self._var = tk.BooleanVar(value=value)
        ttk.Checkbutton(self, text=label, variable=self._var).pack(anchor="w")

    @property
    def var(self) -> tk.BooleanVar:
        return self._var

    def get(self) -> bool:
        return bool(self._var.get())


class LabeledCombobox(ttk.Frame):
    """A labelled read-only ``ttk.Combobox``. ``options`` are ``(label, value)``."""

    def __init__(
        self,
        master: tk.Misc,
        label: str,
        options: Sequence[tuple[str, object]],
    ) -> None:
        super().__init__(master)
        self._choices = list(options)
        self._by_label = {option_label: value for option_label, value in self._choices}

        ttk.Label(self, text=label).pack(side="left", padx=(0, 4))
        self._combo = ttk.Combobox(
            self,
            state="readonly",
            values=[option_label for option_label, _ in self._choices],
        )
        if self._choices:
            self._combo.current(0)
        self._combo.pack(side="left")

    def get(self) -> object:
        """Return the value behind the selected label."""
        return self._by_label.get(self._combo.get())

    def set_enabled(self, enabled: bool) -> None:
        """Enable or disable the combobox (read-only when enabled)."""
        self._combo.configure(state="readonly" if enabled else "disabled")


class LabeledSpinbox(ttk.Frame):
    """A labelled integer ``ttk.Spinbox`` with ``from_``/``to`` bounds."""

    def __init__(
        self,
        master: tk.Misc,
        label: str,
        *,
        from_: int = 1,
        to: int = 1000,
        value: int = 1,
    ) -> None:
        super().__init__(master)
        self._from = from_
        ttk.Label(self, text=label).pack(side="left", padx=(0, 4))
        self._spin = ttk.Spinbox(self, from_=from_, to=to, width=6)
        self._spin.set(value)
        self._spin.pack(side="left")

    def get(self) -> int:
        try:
            return int(self._spin.get())
        except (ValueError, tk.TclError):
            return self._from

    def set_enabled(self, enabled: bool) -> None:
        """Enable or disable the spinbox (used for dependent fields)."""
        self._spin.configure(state="normal" if enabled else "disabled")
