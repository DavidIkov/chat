# Task 02 — api_server HTTP client

**Depends on:** 00, 01. **Feeds into:** 07, 08, 09 (and 11).

## Goal

Implement a stateless, typed client for the `api_server`, porting
`internal/webui_server/services/apiclient` to Python with the standard library
only (`urllib.request`). This is the only place in the client that speaks HTTP.

## Deliverables

| File | Mirror | State |
|------|--------|-------|
| `chat_client/apiclient/paths.py` | `constants.go` | done |
| `chat_client/apiclient/transport.py` | `client.go` (`do`) | **implement** |
| `chat_client/apiclient/users.py` | `users.go` | **implement** |
| `chat_client/apiclient/chats.py` | `chats.go` | **implement** |
| `chat_client/apiclient/client.py` | `client.go` | done (composition) |

## Interface contract

```python
class Transport:
    def __init__(self, timeout: float | None = HTTP_TIMEOUT_SECONDS) -> None
    def request(self, method, base_url, path, *, query=None, token=None, body=None) -> Any
    def close(self) -> None

class UsersMixin:
    def register(self, base_url, name, password) -> UserSession
    def log_in(self, base_url, name, password) -> UserSession
    def log_out(self, base_url, token) -> None
    def delete_user(self, base_url, token, delete_messages: bool) -> None
    def get_users(self, base_url, token, uids: Iterable[int]) -> list[User]

class ChatsMixin:
    def create_chat(self, base_url, token, name) -> int
    def get_chats(self, base_url, token, uids=None) -> list[Chat]
    def get_chat_messages(self, base_url, token, chat_uid, *, limit=0,
                          before_message_uid=0, after_message_uid=0) -> list[Message]
    def send_message(self, base_url, token, chat_uid, text) -> int
    def get_chat_members(self, base_url, token, chat_uid) -> list[Member]
    def create_join_link(self, base_url, token, chat_uid, *,
                         lifetime_seconds=0, max_uses=0) -> JoinLink
    def join_chat(self, base_url, token, join_token) -> int
    def leave_chat(self, base_url, token, chat_uid) -> None

class ApiClient(UsersMixin, ChatsMixin, Transport): ...
```

## Endpoint reference (must match exactly)

Auth header: `Authorization: Bearer <token>` (only when `token` is non-empty).
Bodies are JSON. `query` is `urlencode(..., doseq=True)`; omit it when empty.

| Method | Path | Request body / query | Response → return |
|--------|------|----------------------|-------------------|
| POST | `/user/register` | `{"name","password"}` | `resp["user_session"]` → `UserSession` |
| POST | `/user/login` | `{"name","password"}` | `resp["user_session"]` → `UserSession` |
| POST | `/user/logout` | — (Bearer) | `None` |
| POST | `/user/delete` | `{"delete_messages": bool}` (Bearer) | `None` |
| GET | `/user/get` | `uids=1&uids=2&uids=3` **repeated key, one uid each** | `resp["users"]` → `list[User]` |
| POST | `/chat/create` | `{"name"}` (Bearer) | `resp["chat_uid"]` → `int` |
| GET | `/chat/get_chats` | `uids=1&uids=2` **optional, repeated key** | `resp["chats"]` → `list[Chat]` |
| GET | `/chat/{uid}/get_messages` | `limit`, `before_message_uid`, `after_message_uid` (all optional) | `resp["messages"]` → `list[Message]` |
| POST | `/chat/{uid}/send_message` | `{"text"}` (Bearer) | `resp["message_uid"]` → `int` |
| GET | `/chat/{uid}/get_members` | — (Bearer) | `resp["members"]` → `list[Member]` |
| POST | `/chat/{uid}/create_join_link` | `{"lifetime_seconds","max_uses"}` (Bearer) | `JoinLink` from `token`/`expires_at` |
| POST | `/chat/join_chat` | `{"token"}` (Bearer) | `resp["chat_uid"]` → `int` |
| POST | `/chat/{uid}/leave` | — (Bearer) | `None` |

### Critical details

* **`get_chats` with no uids must send no `uids` query** — the backend then
  returns every chat the caller belongs to (decision D8). Sending
  `uids=` (empty) would break it. `uids=None`/empty ⇒ omit.
* **`get_users` repeats the `uids` key** (`uids=1&uids=2&uids=3`): the api_server
  binds `[]uint` from repeated form values via `go-playground/form/v4`, and the
  Go WebUI client does the same (`apiclient/users.go`). A single comma-separated
  value (`uids=1,2,3`) is **not** parsed and yields a 400. `get_chats` uses the
  same repeated-key encoding when a uid list is supplied.
* **`get_chat_messages` omits zero-valued parameters** (`limit=0`,
  `before=0`, `after=0`), matching the Go client.
* Missing list keys (`chats`, `messages`, `members`, `users`) must yield `[]`.
* `limit`/`lifetime_seconds`/`max_uses`/`before`/`after` are omitted when `0`.

## Error handling

`request` normalises every failure to `APIError` (from `chat_client.errors`):

1. Read the response body; try `json.loads` (non-JSON is fine).
2. If the body has `{"error": {"field", "message"}}` →
   `APIError(message, status=<code>, field=<field>)`. This covers the backend's
   habit of returning a 2xx body that still carries an error object.
3. Else if status is not 2xx → `APIError(body_text or HTTP reason, status=...)`.
4. Network failures (`urllib.error.URLError`, socket timeouts, TLS errors,
   `OSError`) → `APIError("Could not reach <base_url>: <reason>", status=None)`.
5. Empty body → `None`; otherwise the parsed JSON.

Use `urllib.request.Request`, set `Content-Type: application/json` only when a
body is sent, and pass `timeout` from the transport.

## Steps

1. Implement `Transport.request` (URL join, query encoding, JSON encode/decode,
   bearer header, error mapping). Right-strip `/` from `base_url`.
2. Implement `UsersMixin` methods, returning `models`.
3. Implement `ChatsMixin` methods, formatting `%d` paths with `chat_uid`.
4. Keep `ApiClient` a no-op composition of the three.

## Acceptance criteria

- [ ] Against a running `api_server`: register → login → get_users round-trips.
- [ ] `get_chats(base, token)` sends no `uids`; `get_chats(base, token, [5])`
      sends `uids=5`.
- [ ] A backend validation error surfaces as `APIError` with `.field` and
      `.message`.
- [ ] An unreachable URL surfaces as `APIError` with `status is None` and does not
      raise `URLError`.
- [ ] `get_chat_messages(..., limit=0)` does not include a `limit` parameter.
- [ ] No third-party imports; works on Python 3.10+.

## Out of scope

Retries, connection pooling tuning, TLS configuration, WebSockets.
Callers (controller tasks) own timeouts/UX.

## References

- `internal/webui_server/services/apiclient/{client,users,chats,constants}.go`
- `internal/shared/api/{user,chat}/*.go`
- `docs/architecture.md` § 4, § 7.
