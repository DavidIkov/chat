"""Chat desktop GUI client.

A native Tkinter front end for one or more ``api_server`` instances. It mirrors
the behaviour of ``internal/webui_server`` (same HTTP API, same features: server
connections, login/register, chats, messages, members, join links, account
deletion) but runs as a local desktop application.

Because it is a desktop app rather than a proxy served to a browser, two things
differ from the Go WebUI:

* bearer tokens are kept in the user's own config file (``0600``) instead of a
  server-side session;
* there is no ``html/template``/PRG layer — an in-process router swaps Tk frames
  and blocking HTTP runs on a background thread pool so the UI never freezes.

See ``gui_client/docs/architecture.md`` for the full module map and
``gui_client/docs/tasks/`` for the implementation breakdown.
"""

from __future__ import annotations

__all__ = ["__version__"]

__version__ = "0.1.0"
