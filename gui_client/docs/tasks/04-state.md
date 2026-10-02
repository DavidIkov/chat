# Task 04 — App state & persistence

**Depends on:** 00, 01, 03. **Feeds into:** 07, 08, 09, 11.

## Goal

Port `internal/webui_server/services/sessions` to a single-user desktop model:
one `AppState`, a list of `ServerConnection`s each with its own `UserSession`, and
a one-shot `PendingJoinLink`. Add the serialisation bridge to the config schema
(task 03).

## Deliverables

| File | Mirror | State |
|------|--------|-------|
| `chat_client/state/session.py` | `services/sessions/{types,service,errors}.go` | **implement** |
| `chat_client/state/persistence.py` | *(new)* | **implement** |
| `chat_client/state/__init__.py` | — | done |

## Interface contract

```python
@dataclass
class ServerConnection:
    id: int; url: str; name: str = ""; session: UserSession | None = None
    @property label -> str          # name or url
    @property signed_in -> bool

@dataclass
class PendingJoinLink:
    server_id: int; chat_uid: int; token: str

@dataclass
class AppState:
    connections: list[ServerConnection] = <empty>
    next_id: int = 0
    pending_join_link: PendingJoinLink | None = None

    def add_server(self, url: str, name: str = "") -> ServerConnection
    def server_by_id(self, server_id: int) -> ServerConnection      # raises ServerNotFoundError
    def find_server(self, server_id: int) -> ServerConnection | None
    def remove_server(self, server_id: int) -> None
    def list_servers(self) -> list[ServerConnection]                # shallow copy
    def set_server_session(self, server_id, session: UserSession | None) -> None
    def clear_server_session(self, server_id: int) -> None
    def set_pending_join_link(self, server_id, chat_uid, token) -> None
    def take_pending_join_link(self, server_id, chat_uid) -> str

def validate_server_url(url: str) -> str                            # raises InvalidServerUrlError
def state_to_config(state: AppState) -> dict
def state_from_config(data: dict) -> AppState
```

## Behaviour (port of the Go service)

* `validate_server_url`: `urlparse`; scheme must be `http`/`https` and host
  non-empty; **strip trailing `/`**; raise `InvalidServerUrlError` otherwise.
  (Go: `AddServer` in `sessions/service.go`.)
* `add_server`: validate, increment `next_id` **first**, append
  `ServerConnection(id=next_id, url=normalised, name=name)`, return it.
* `server_by_id`: linear search; raise `ServerNotFoundError(server_id)`.
* `remove_server`: drop the entry (and its token) by id; no-op if absent.
* `list_servers`: return a **copy** of the list so callers can render freely.
* `set_server_session` (and `clear_server_session` → `set(..., None)`): raise
  `ServerNotFoundError` when the id is unknown.
* `take_pending_join_link`: return `""` unless the pending link matches **both**
  `server_id` and `chat_uid`; when it matches, clear it and return the token.
  Matching on both is what the Go `TakePendingJoinLink` does.
* Threading: no locks — `AppState` is main-thread only (see
  `architecture.md` § 3). Say so in the module docstring.

### `persistence.py`

* `state_to_config`: `{"version": CONFIG_VERSION, "next_id": state.next_id,
  "connections": [...]}`; each connection `{"id", "url", "name",
  "session": session.to_dict() or None}`.
* `state_from_config`: tolerant reader — missing/partial doc ⇒ empty `AppState`;
  skip malformed connection entries instead of raising; coerce ids to int;
  restore `next_id` (and if absent, derive it from the max connection id so new
  ids never collide).

## Acceptance criteria

- [ ] `add_server("http://x:1/")` stores url without the trailing slash and
      assigns id 1, then 2.
- [ ] `add_server("ftp://x")` raises `InvalidServerUrlError`.
- [ ] `server_by_id(99)` raises `ServerNotFoundError`.
- [ ] `set_server_session(1, UserSession(7, "t"))` then `find_server(1).signed_in`
      is `True`; `label` falls back to the URL when `name` is empty.
- [ ] `take_pending_join_link` returns the token once, then `""`.
- [ ] `state_from_config(state_to_config(s))` preserves connections, sessions,
      names and `next_id`.
- [ ] `state_from_config({})` and `state_from_config({"connections": [{}]})` do
      not raise.
- [ ] No `NotImplementedError` remains in `state/`.

## Out of scope

Writing to disk (task 03 owns `ConfigStore`); any UI.

## References

- `internal/webui_server/services/sessions/{types,service,errors,constructor}.go`
- `docs/architecture.md` § 4, § 5.
