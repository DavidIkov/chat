# Task 07 — Server connections feature

**Depends on:** 04 (state), 06 (shell/widgets/dialogs). **Feeds into:** 10.

## Goal

Implement the landing page and one connection's page, plus their controller
operations: FR-1 (add), FR-2 (list), FR-3 (remove), FR-4 (login/register),
FR-5 (logout) and FR-15 (delete account). Port of
`internal/webui_server/handlers/servers`.

## Deliverables

| File | Mirror | Implement |
|------|--------|-----------|
| `chat_client/controller/servers.py` | `handlers/servers/handlers.go` | `ServersMixin` |
| `chat_client/ui/views/servers.py` | `templates/index.html` | `ServersView` |
| `chat_client/ui/views/server.py` | `templates/server.html` | `ServerView` |

## 7.1 `ServersMixin` (controller)

All methods assume `self.state`, `self.api`, `self.tasks`, `self.config` and the
`ControllerBase` helpers exist (task 06).

| Method | api call | Success | Failure |
|--------|----------|---------|---------|
| `add_server(url, name)` | none | `state.add_server`, `_persist()`, `show_server(id)` | `InvalidServerUrlError` → view banner |
| `remove_server(server_id)` | none | `state.remove_server`, `_persist()`, `show_servers()` | — |
| `log_in(server_id, name, password)` | `api.log_in` | `state.set_server_session`, `_persist()`, `show_server(id)` | banner (401/422) |
| `register(server_id, name, password)` | `api.register` | as `log_in` | banner |
| `log_out(server_id)` | `api.log_out` | `state.clear_server_session`, `_persist()`, `show_server(id)` | ignore the backend error, still clear locally |
| `delete_account(server_id, delete_messages)` | `api.delete_user` | clear session, `_persist()`, `show_server(id, notice="The account was deleted on this server.")` | banner |

Details:

* Resolve the connection with `self._server(server_id)` / `self._signed_in_server`;
  a missing id → error view, a signed-out connection on delete → `show_server`.
* `add_server` may `ValidationError` locally for an empty URL; the invalid-scheme
  case raises `InvalidServerUrlError` from `state.add_server`.
* Never call the backend from `add_server`/`remove_server` (parity with FR-1/FR-3).
* `delete_account` must clear the local session even though the token is now dead
  on the backend (the WebUI does exactly this).
* Failure messages go to the current view's banner via `_show_api_error`; do not
  navigate away (so the user can retry). For `log_in`/`register` the WebUI answers
  401/422 — the banner text is the backend message.

## 7.2 `ServersView` (`ui/views/servers.py`)

```python
class ServersView(BaseView):
    def on_show(self) -> None                      # render state.list_servers()
    def render_servers(self, servers) -> None
```

Layout:

1. `Banner` (from `BaseView`).
2. **Add-an-api-server** fieldset: `LabeledEntry("URL", placeholder="http://localhost:8080")`,
   `LabeledEntry("Name", placeholder="optional")`, an **Add server** button →
   `self.app.controller.add_server(url.get(), name.get())`; clear the fields on
   success is optional (navigation happens).
3. **Connection list** (or "No servers yet. Add one above."):
   each row shows the `label`, the `url` as muted text, a **signed in** marker when
   `connection.signed_in`, an **Open** button → `controller.show_server(id)`, and a
   **Remove** button → `controller.remove_server(id)`.

`on_show` does **no** network call — it renders `self.app.state.list_servers()`.

## 7.3 `ServerView` (`ui/views/server.py`)

```python
class ServerView(BaseView):
    def on_show(self) -> None                      # resolve route.server_id
    def render(self, connection) -> None
```

`on_show` uses `state.find_server(route.server_id)`; when `None` it calls
`controller.show_error("server not found")`. Then `render(connection)`:

* A header with the connection `label` and a **← All servers** button.
* If `route.notice` is non-empty → `banner.show_notice(route.notice)`; otherwise
  `banner.clear()`.
* Signed **out** (`not connection.signed_in`) — two forms side by side
  (`ttk.LabelFrame`s), each with Name + Password (`LabeledEntry(..., show="•")`):
  * **Log in** → `controller.log_in(id, name, password)`
  * **Register** → `controller.register(id, name, password)`
* Signed **in**:
  * "Signed in as uid {connection.session.uid}."
  * **Open chats** button → `controller.show_chats(id)`.
  * **Log out** button → `controller.log_out(id)`.
  * **Delete account** fieldset with a `LabeledCheckbutton("Also delete my messages
    from every chat")` and a danger-styled **Delete account** button. The button
    first calls `dialogs.confirm_delete_account(self, connection.label)`; only when
    confirmed does it call
    `controller.delete_account(id, delete_messages)`.

Avoid putting passwords into any log output.

## Acceptance criteria

- [ ] Add `http://localhost:8080` → the connection appears and its page opens.
- [ ] A bad URL (`ftp://x`, empty) shows an error banner and adds nothing.
- [ ] Register then log out then log in with the same credentials all work and the
      page reflects the signed-in state (uid shown).
- [ ] Remove returns to the landing page and the connection is gone.
- [ ] Delete account (with and without the checkbox) clears the session and shows
      the "account was deleted" notice.
- [ ] Every action persists to the config file (verify `state` after each).
- [ ] While `log_in`/`register` is in flight the window stays responsive and the
      form is busy.

## Out of scope

Chat pages (08/09). Editing a connection's URL/name.

## References

- `internal/webui_server/handlers/servers/{handlers,types,constructor,constants}.go`
- `internal/webui_server/templates/{index,server}.html`
- `docs/architecture.md` § 6.
