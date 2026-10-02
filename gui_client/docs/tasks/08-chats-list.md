# Task 08 — Chat list feature

**Depends on:** 04 (state), 06 (shell/widgets). **Feeds into:** 10.

## Goal

Implement FR-6 (list the caller's chats), FR-7 (create a chat) and FR-8 (join a
chat by token), with the chats page. Port of the list/create/join handlers in
`internal/webui_server/handlers/chats` and `templates/chats.html`.

## Deliverables

| File | Mirror | Implement |
|------|--------|-----------|
| `chat_client/controller/chats.py` | `handlers/chats/handlers.go` (list/create/join) | `ChatsMixin` |
| `chat_client/ui/views/chats.py` | `templates/chats.html` | `ChatsView` |

## 8.1 `ChatsMixin` (controller)

```python
def load_chats(self, server_id: int) -> None
def create_chat(self, server_id: int, name: str) -> None
def join_chat(self, server_id: int, token: str) -> None
```

| Method | api call | Success | Failure |
|--------|----------|---------|---------|
| `load_chats` | `api.get_chats(url, token)` — **no uids** | `view.render_chats(chats)` | banner (or error view on non-API failure) |
| `create_chat` | `api.create_chat(url, token, name)` | `show_chat(server_id, new_uid)` | banner, stay on chats |
| `join_chat` | `api.join_chat(url, token, join_token)` | `show_chat(server_id, new_uid)` | banner with the backend message (invalid/expired/exhausted token) |

Details:

* Guard with `self._signed_in_server(server_id)`; when `None` the helper has
  already navigated (to the server page or the error view) — return.
* `get_chats` must be called with `uids=None` so the backend returns every chat
  the caller belongs to (D8). Never pass an empty list.
* Local validation before scheduling: chat name must be 4–32 chars
  (`constants.CHAT_NAME_MIN/MAX`) and the join token non-empty; on failure show a
  `ValidationError` message and skip the request.
* `load_chats` should call `view.set_loading(True/False)` around the request.
* On a `load_chats` failure the WebUI renders the chats page with the error; do
  the same via the view's banner when the view is mounted, else `show_error`.

## 8.2 `ChatsView` (`ui/views/chats.py`)

```python
class ChatsView(BaseView):
    def on_show(self) -> None                       # controller.load_chats(route.server_id)
    def render_chats(self, chats: list[Chat]) -> None
    def show_empty(self) -> None
```

`on_show` resolves the connection for a header label (via `app.state`) and calls
`self.app.controller.load_chats(self.route.server_id)`. It must not perform I/O
itself.

Layout:

1. `Banner`.
2. Header "Chats — {connection.label}" with **← Server** (→ `show_server(id)`) and a
   **Refresh** button (→ `controller.load_chats(id)` / `controller.refresh()`).
3. **Create chat** fieldset: `LabeledEntry("Name")` + **Create** →
   `controller.create_chat(id, name.get())`.
4. **Join chat** fieldset: `LabeledEntry("Token")` + **Join** →
   `controller.join_chat(id, token.get())`.
5. Chat list: rows are buttons/links labelled with `chat.name`, each →
   `controller.show_chat(id, chat.chat_uid)`. Empty list → `show_empty()`
   ("No chats yet."). Show a "Loading…" state until `render_chats`/`show_empty` is
   called.

`render_chats` must clear any previously rendered rows first (the view may be
refreshed in place).

## Acceptance criteria

- [ ] Opening the page lists every chat the signed-in user belongs to.
- [ ] A chat created outside this client also appears (proves the no-`uids` call).
- [ ] Creating a chat with a valid name (4–32) opens it; an invalid name shows a
      local validation message and makes no request.
- [ ] Joining with a valid token opens the chat; an invalid token shows the backend
      message without leaving the page.
- [ ] Refresh re-fetches the list; the window stays responsive during the call.
- [ ] Visiting the page while signed out redirects to that connection's page.

## Out of scope

The chat page itself (task 09).

## References

- `internal/webui_server/handlers/chats/handlers.go`
  (`ChatsPageHandler`, `CreateChatHandler`, `JoinChatHandler`, `chatsOrNil`)
- `internal/webui_server/templates/chats.html`
- `docs/architecture.md` § 6.
