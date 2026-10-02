"""Modal dialogs.

The WebUI inlines almost every form and only needs a confirmation for account
deletion; the desktop client keeps that spirit but uses native modal dialogs
where a page-based flow would be awkward. Anything that can live inline (add
server, login/register, create/join chat, join link) does, matching the templates.

Windows created here are ``transient`` + ``grab_set`` so they behave modally.
"""

from __future__ import annotations

import tkinter as tk
from tkinter import messagebox, ttk


def show_info(parent: tk.Misc, title: str, message: str) -> None:
    """Show an informational message box."""
    messagebox.showinfo(title, message, parent=parent)


def show_error(parent: tk.Misc, title: str, message: str) -> None:
    """Show an error message box."""
    messagebox.showerror(title, message, parent=parent)


def confirm(parent: tk.Misc, title: str, message: str) -> bool:
    """Yes/No confirmation. Returns ``True`` when the user confirms."""
    return bool(messagebox.askyesno(title, message, parent=parent))


def confirm_delete_account(
    parent: tk.Misc, server_label: str
) -> tuple[bool, bool]:
    """Confirm permanent account deletion.

    Returns ``(confirmed, delete_messages)`` where ``delete_messages`` reflects
    the "also delete my messages from every chat" checkbox (FR-15). A custom
    dialog is needed because :func:`confirm` cannot carry a checkbox.
    """
    confirmed = False
    delete_messages = False

    dialog = tk.Toplevel(parent)
    dialog.title("Delete account")
    dialog.transient(parent.winfo_toplevel())
    dialog.resizable(False, False)

    body = ttk.Frame(dialog, padding=12)
    body.pack(fill="both", expand=True)

    ttk.Label(
        body,
        text=f"Permanently delete your account on {server_label}?",
        wraplength=420,
        justify="left",
    ).pack(anchor="w")
    ttk.Label(
        body,
        text="This cannot be undone.",
        foreground="crimson",
    ).pack(anchor="w", pady=(4, 8))

    delete_var = tk.BooleanVar(value=False)
    ttk.Checkbutton(
        body,
        text="Also delete my messages from every chat",
        variable=delete_var,
    ).pack(anchor="w")

    buttons = ttk.Frame(body)
    buttons.pack(anchor="e", pady=(12, 0))

    def on_cancel() -> None:
        dialog.destroy()

    def on_delete() -> None:
        nonlocal confirmed, delete_messages
        confirmed = True
        delete_messages = bool(delete_var.get())
        dialog.destroy()

    style = ttk.Style(dialog)
    style.configure("Danger.TButton", foreground="crimson")
    ttk.Button(
        buttons, text="Delete account", style="Danger.TButton", command=on_delete
    ).pack(side="right")
    ttk.Button(buttons, text="Cancel", command=on_cancel).pack(
        side="right", padx=(0, 6)
    )

    dialog.bind("<Escape>", lambda _event: on_cancel())
    dialog.protocol("WM_DELETE_WINDOW", on_cancel)

    _center_on_parent(dialog, parent)
    dialog.grab_set()
    dialog.focus_set()
    parent.wait_window(dialog)
    return confirmed, delete_messages


def _center_on_parent(dialog: tk.Toplevel, parent: tk.Misc) -> None:
    """Position ``dialog`` over the centre of ``parent``."""
    dialog.update_idletasks()
    try:
        px = parent.winfo_rootx()
        py = parent.winfo_rooty()
        pw = parent.winfo_width()
        ph = parent.winfo_height()
    except tk.TclError:  # pragma: no cover - parent already destroyed
        return
    width = dialog.winfo_width()
    height = dialog.winfo_height()
    x = px + max(0, (pw - width) // 2)
    y = py + max(0, (ph - height) // 2)
    dialog.geometry(f"+{x}+{y}")
