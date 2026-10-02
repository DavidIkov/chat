"""Application-wide constants.

Values that describe the *backend contract* (validator limits, pagination
defaults, join-link presets) are deliberately duplicated from the Go servers so
this package stays a dependency-free, standalone Python project. Keep them in
sync with:

* ``internal/shared/api/validator/constants.go`` — length limits;
* ``internal/webui_server/templates/chat.html`` — join-link presets.
"""

from __future__ import annotations

# --- identity ---------------------------------------------------------------

APP_NAME = "Chat"
WINDOW_TITLE = "Chat"
#: ``$XDG_CONFIG_HOME/<CONFIG_DIR_NAME>/<CONFIG_FILE_NAME>`` (or ``~/.config/...``).
CONFIG_DIR_NAME = "chat"
CONFIG_FILE_NAME = "gui_client.json"
CONFIG_FILE_MODE = 0o600
CONFIG_DIR_MODE = 0o700

#: Bumped whenever the on-disk schema changes; see ``config/store.py``.
CONFIG_VERSION = 1

# --- backend validation limits (mirror internal/shared/api/validator) -------

USER_NAME_MIN, USER_NAME_MAX = 4, 16
USER_PASSWORD_MIN, USER_PASSWORD_MAX = 4, 16
CHAT_NAME_MIN, CHAT_NAME_MAX = 4, 32
MESSAGE_TEXT_MIN, MESSAGE_TEXT_MAX = 1, 256

#: ``GET /chat/{uid}/get_messages`` returns the api_server default (50) when the
#: limit is 0, so the client sends no limit and lets the backend decide.
MESSAGE_PAGE_LIMIT = 0

# --- networking / concurrency ----------------------------------------------

HTTP_TIMEOUT_SECONDS = 30.0
#: How often the main loop drains the completed-task queue (milliseconds).
TASK_POLL_INTERVAL_MS = 40
DEFAULT_WORKER_THREADS = 4

# --- user interface ---------------------------------------------------------

DEFAULT_WINDOW_SIZE = (980, 700)
MIN_WINDOW_SIZE = (640, 480)
#: Shown in the status bar; mirrors the WebUI footer hint (FR-13).
REFRESH_HINT = "Refresh (F5) to load new data — updates are not live."

#: ``(label, seconds)`` pairs offered by the join-link form (FR-11).
JOIN_LINK_LIFETIME_OPTIONS = (
    ("1 hour", 3600),
    ("1 day", 86400),
    ("7 days", 604800),
    ("30 days", 2592000),
)
JOIN_LINK_DEFAULT_MAX_USES = 10
