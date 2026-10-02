"""Console entry point: ``python -m chat_client``.

Bootstraps the object graph exactly once and hands control to Tk:

1. parse ``--config`` (and ``--verbose``);
2. build :class:`~chat_client.config.store.ConfigStore` and load
   :class:`~chat_client.state.session.AppState` from it (a malformed file is
   reported and a fresh state is used);
3. build the :class:`~chat_client.apiclient.client.ApiClient`;
4. construct :class:`~chat_client.ui.app.App`, which wires the task runner and
   controller and shows the landing page;
5. run ``mainloop``.

The bootstrap is the client's analogue of ``cmd/webui_server/main.go``.
"""

from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path


def build_parser() -> argparse.ArgumentParser:
    """Return the CLI parser (``--config``, ``--verbose``)."""
    parser = argparse.ArgumentParser(
        prog="chat-client",
        description="Native Tkinter desktop client for the Chat api_server.",
    )
    parser.add_argument(
        "--config",
        type=Path,
        default=None,
        metavar="PATH",
        help=(
            "path to the JSON config file "
            "(default: $XDG_CONFIG_HOME/chat/gui_client.json)"
        ),
    )
    parser.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="enable debug logging",
    )
    return parser


def _warn_config_error(exc: Exception) -> None:
    """Report a malformed config file without ever showing a traceback.

    Prefers a native warning dialog; when no display is available (or Tk is
    otherwise unusable) the same message goes to ``stderr`` instead.
    """
    message = (
        f"Could not read the configuration file:\n{exc}\n\n"
        "The client will start with an empty configuration."
    )
    logging.warning("config error: %s", exc)
    try:
        import tkinter as tk
        from tkinter import messagebox

        root = tk.Tk()
        root.withdraw()
        messagebox.showwarning("Chat", message)
        root.destroy()
    except Exception:  # noqa: BLE001 - any Tk failure just means "no display"
        print(f"chat-client: {message}", file=sys.stderr)


def main(argv: list[str] | None = None) -> int:
    """Run the GUI client. Returns a process exit code."""
    args = build_parser().parse_args(argv)
    logging.basicConfig(level=logging.DEBUG if args.verbose else logging.INFO)

    # Imported lazily so ``import chat_client.__main__`` and ``--help`` work
    # headlessly, without constructing any widgets or opening a display.
    try:
        import tkinter as tk
    except ImportError as exc:  # pragma: no cover - Tkinter is a hard requirement
        print(f"chat-client: Tkinter is not available: {exc}", file=sys.stderr)
        return 1

    from chat_client.apiclient.client import ApiClient
    from chat_client.config.store import ConfigStore
    from chat_client.errors import ConfigError
    from chat_client.state.persistence import state_from_config
    from chat_client.ui.app import App

    config = ConfigStore(args.config)
    try:
        data = config.load()
    except ConfigError as exc:
        # A corrupt file must not abort startup: warn and begin fresh.
        _warn_config_error(exc)
        data = {}

    state = state_from_config(data)
    api = ApiClient()

    try:
        app = App(state, api, config)
    except tk.TclError as exc:
        print(
            f"chat-client: cannot open a window: {exc}\n"
            "A graphical display is required; check that DISPLAY (or "
            "WAYLAND_DISPLAY) is set.",
            file=sys.stderr,
        )
        return 1

    app.mainloop()
    return 0


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main())
