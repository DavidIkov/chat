"""In-memory application state, mirroring ``services/sessions``.

The Go WebUI keeps a ``WebUISession`` per browser cookie plus a
``ServerConnection`` per added api_server. A desktop client is single-user, so
there is exactly one :class:`AppState` and it holds the connection list directly.

Threading model (important): **all reads and writes of ``AppState`` happen on the
Tk main thread.** Background work (HTTP calls) runs in
:class:`~chat_client.tasks.runner.TaskRunner` and only ever receives plain
values (``base_url``, ``token``, uids); it never touches ``AppState``. That is
why the methods below take no locks — the Go version's mutexes exist to guard
against the HTTP server's goroutines, which have no equivalent here. See the
client architecture doc, § "Threading model".
"""

from __future__ import annotations

from dataclasses import dataclass, field
from urllib.parse import urlparse

from chat_client.errors import InvalidServerUrlError, ServerNotFoundError
from chat_client.models.user import UserSession


@dataclass
class ServerConnection:
    """One api_server instance added by the user.

    Mirrors ``sessions.ServerConnection``: ``id`` is a client-local, stable id
    (the WebUI's ``{server_id}``); ``session`` is ``None`` until the user signs in
    and is per-connection so different connections can be different users.
    """

    id: int
    url: str
    name: str = ""
    session: UserSession | None = None

    @property
    def label(self) -> str:
        """Display name: the user's label, falling back to the URL."""
        return self.name or self.url

    @property
    def signed_in(self) -> bool:
        return self.session is not None


@dataclass
class PendingJoinLink:
    """A freshly minted join token, shown once on the next chat view render.

    Mirrors ``sessions.pendingJoinLink`` + ``SetPendingJoinLink`` /
    ``TakePendingJoinLink``: the join-link action stores the token here and then
    navigates, so the POST-equivalent never renders its own result and a refresh
    cannot re-mint a token.
    """

    server_id: int
    chat_uid: int
    token: str


@dataclass
class AppState:
    """The single, mutable root of client state."""

    connections: list[ServerConnection] = field(default_factory=list)
    next_id: int = 0
    pending_join_link: PendingJoinLink | None = None

    # --- connections (FR-1..3) ---------------------------------------------

    def add_server(self, url: str, name: str = "") -> ServerConnection:
        """Validate ``url``, append a connection and return it (FR-1).

        :class:`~chat_client.errors.InvalidServerUrlError` unless the URL
        parses and its scheme is ``http``/``https`` with a non-empty host. The
        stored URL is right-stripped of ``/``. ``next_id`` is incremented.
        """
        normalised = validate_server_url(url)
        # Increment first so ids start at 1, mirroring WebUISession.AddServer.
        self.next_id += 1
        connection = ServerConnection(id=self.next_id, url=normalised, name=name)
        self.connections.append(connection)
        return connection

    def server_by_id(self, server_id: int) -> ServerConnection:
        """Return the connection with ``server_id`` or raise
        :class:`~chat_client.errors.ServerNotFoundError`."""
        connection = self.find_server(server_id)
        if connection is None:
            raise ServerNotFoundError(server_id)
        return connection

    def find_server(self, server_id: int) -> ServerConnection | None:
        """Return the connection with ``server_id``, or ``None``."""
        for connection in self.connections:
            if connection.id == server_id:
                return connection
        return None

    def remove_server(self, server_id: int) -> None:
        """Drop a connection and its stored credentials (FR-3)."""
        for index, connection in enumerate(self.connections):
            if connection.id == server_id:
                del self.connections[index]
                return

    def list_servers(self) -> list[ServerConnection]:
        """Return a shallow copy of the connection list for rendering."""
        return list(self.connections)

    # --- per-connection session (FR-4/5/14/15) -----------------------------

    def set_server_session(self, server_id: int, session: UserSession | None) -> None:
        """Store (or clear, with ``None``) the api_server session on a connection."""
        self.server_by_id(server_id).session = session

    def clear_server_session(self, server_id: int) -> None:
        """Forget the api_server session on a connection (used by logout/delete)."""
        self.set_server_session(server_id, None)

    # --- one-shot join link (FR-11) ----------------------------------------

    def set_pending_join_link(self, server_id: int, chat_uid: int, token: str) -> None:
        """Stash a freshly created join token for the next chat render."""
        self.pending_join_link = PendingJoinLink(
            server_id=server_id, chat_uid=chat_uid, token=token
        )

    def take_pending_join_link(self, server_id: int, chat_uid: int) -> str:
        """Return and clear the pending token for ``(server_id, chat_uid)``.

        Returns ``""`` when none is pending. Tokens belonging to another
        connection or chat are left untouched, matching the Go implementation.
        """
        pending = self.pending_join_link
        if (
            pending is None
            or pending.server_id != server_id
            or pending.chat_uid != chat_uid
        ):
            return ""
        self.pending_join_link = None
        return pending.token


def validate_server_url(url: str) -> str:
    """Validate and normalise a user-supplied api_server URL.

    Returns the normalised URL (``/`` stripped) or raises
    :class:`~chat_client.errors.InvalidServerUrlError`. Kept module-level so the
    add-server form can validate without constructing a connection.
    """
    try:
        parsed = urlparse(url)
    except ValueError as exc:  # e.g. a malformed IPv6 literal
        raise InvalidServerUrlError() from exc
    if parsed.scheme not in ("http", "https") or not parsed.hostname:
        raise InvalidServerUrlError()
    return url.rstrip("/")
