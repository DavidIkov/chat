# GUI Client — Architecture

Companion to [`requirements.md`](requirements.md). Explains how the Python client
is structured, and how each layer maps onto the Go `webui_server` so the two can
be read side by side. The per-task implementation instructions live in
[`tasks/`](tasks/README.md).

---

## 1. Overview and system context

```
        Tk widgets                      JSON over HTTP
  ┌────────────────────┐          ┌──────────────────────────────┐
  │  chat_client      │ ───────▶ │ api_server A  (Bearer token) │
  │  (desktop process) │ ◀─────── │  + Postgres                  │
  │                    │          └──────────────────────────────┘
  │  • AppState        │  Bearer  ┌──────────────────────────────┐
  │  • ApiClient       │ ───────▶ │ api_server B  (Bearer token) │
  │  • worker pool     │ ◀─────── │  + Postgres                  │
  └────────────────────┘          └──────────────────────────────┘
        │        ▲
   reads/writes  │ results via queue + Tk `after`
        ▼        │
  config.json (0600)
```

Differences from the Go WebUI (which proxies and renders HTML):

1. There is **no browser**: the desktop app holds the tokens itself (C1) and talks
   to api_servers directly (C2).
2. There is **no HTTP server inside the client**: navigation is an in-process
   router (C6).
3. State is **persisted** to a `0600` JSON file (C3) rather than being in-memory.

Everything else — the JSON contract, the multi-server isolation, the flow of each
feature — is the same.

---

## 2. Layering

```
ui/            Tkinter views + widgets        (≈ templates/ + handlers/render)
controller/    one method per user action      (≈ handlers/servers + handlers/chats)
tasks/         worker pool, Tk-safe callbacks  (≈ goroutine per request)
state/         AppState, ServerConnection      (≈ services/sessions)
config/        JSON file I/O                   (≈ new: persistence)
apiclient/     JSON-over-HTTP client           (≈ services/apiclient)
models/        dataclasses                     (≈ internal/shared/api)
```

Dependencies point **downward only**:

```
ui → controller → {state, apiclient, config, tasks, models}
controller → ui.route         (navigation values only)
state → models, errors
apiclient → models, errors, constants
models → (nothing)
ui.route → (nothing)
```

`controller` imports `ui.app` only under `TYPE_CHECKING`; `ui.app` imports the
controller and every view at runtime (it is the composition root). This keeps the
import graph acyclic, mirroring why the Go side keeps `render`/`routes` as leaf
packages.

---

## 3. Threading model (important)

Tkinter is single-threaded and not thread-safe. Network calls must not block it.
The rule is:

* **`AppState` and every widget are touched only on the Tk main thread.**
* **Worker threads only compute.** A task receives plain values
  (`base_url`, `token`, `chat_uid`, `text`, …), calls `ApiClient`, and returns a
  value or raises. It never reads or writes `AppState`.
* `TaskRunner.submit(work, on_success=…, on_error=…)` runs `work` on a
  `ThreadPoolExecutor`, pushes the outcome onto a `queue.Queue`, and a periodic
  `widget.after()` poll drains the queue on the main thread and invokes the
  callbacks there.

Because of this rule, `AppState` needs no locks. (The Go `sessions` service uses
mutexes because its state is shared across HTTP goroutines; the desktop client has
no equivalent concurrent reader.)

Sequence for "send message":

```
view:  on_send() ─▶ controller.send_message(server_id, chat_uid, text)
controller: read token from AppState (main thread)
            tasks.submit(work=lambda: api.send_message(url, token, uid, text),
                         on_success=lambda _: self.load_chat(server_id, uid),
                         on_error=self._show_api_error)
worker:    urllib POST /chat/{uid}/send_message
main:      after() drain → on_success → load_chat → tasks.submit(get_chats/…) …
```

---

## 4. Package responsibilities

### `chat_client.models`
Frozen dataclasses mirroring `internal/shared/api`: `User`, `UserSession`, `Chat`,
`Message`, `Member`, `JoinLink`, plus `Uid`/`TimestampMs` aliases. Each response
type has a `from_dict` classmethod that tolerates missing keys. Request bodies are
plain `dict`s built in `apiclient`.

### `chat_client.apiclient`
Mirrors `services/apiclient`:

* `paths.py` — endpoint constants (`constants.go`).
* `transport.py` — `Transport.request(...)`: URL building, JSON encode/decode,
  `Authorization: Bearer …`, and normalisation of `{"error":{…}}` / non-2xx /
  network failures into `APIError`. Equivalent of `do` in `client.go`.
* `users.py` / `chats.py` — one method per endpoint, composed as mixins.
* `client.py` — `class ApiClient(UsersMixin, ChatsMixin, Transport)`.

The client is **stateless** with respect to connections: every method takes
`base_url` and `token` explicitly, so one instance serves every connection and is
safe to call from worker threads.

### `chat_client.config`
`ConfigStore.load()/save()` with an atomic write (temp file + `os.replace`) and
mode `0600`; `default_config_path()` honours `XDG_CONFIG_HOME`. Schema in § 5.

### `chat_client.state`
`AppState` + `ServerConnection` + `PendingJoinLink`, mirroring
`services/sessions`. `add_server` performs the same URL validation as Go
(`http`/`https`, non-empty host, trailing `/` stripped);
`set/take_pending_join_link` reproduce `SetPendingJoinLink`/`TakePendingJoinLink`
so a join token is shown exactly once. `persistence.py` maps `AppState` ⇄ config
JSON.

### `chat_client.tasks`
`TaskRunner` — see § 3.

### `chat_client.controller`
One method per user action. Each performs the same six steps as a Go handler:
validate input → resolve the `ServerConnection` → call the api client through
`TaskRunner` → on success mutate `AppState`/persist and navigate → on `APIError`
show it in the active view → on unexpected errors show the error view. Navigation
replaces the WebUI's `303` redirects; inline error display replaces "re-render the
page with a message".

### `chat_client.ui`
* `app.py` — `App(tk.Tk)`: window, menu, status bar, and the router
  (`navigate(Route)` destroys the current view and mounts the next). It is the
  composition root: it builds `TaskRunner` and `Controller`.
* `route.py` — `Route`/`RouteName`.
* `views/` — one `BaseView` subclass per page; dumb (render + forward intent).
* `widgets/` — `Banner`, `ScrollableFrame`, `MessageList`, `MemberList`, and the
  labelled form fields.
* `dialogs.py` — modal confirmations (account deletion) and message boxes.

---

## 5. Config schema

`$XDG_CONFIG_HOME/chat/gui_client.json` (or `~/.config/...`), mode `0600`:

```json
{
  "version": 1,
  "next_id": 2,
  "connections": [
    {
      "id": 1,
      "url": "http://localhost:8080",
      "name": "Local",
      "session": { "uid": 1, "token": "…" }
    },
    {
      "id": 2,
      "url": "https://chat.example.com",
      "name": "",
      "session": null
    }
  ]
}
```

Rules:

* Missing file → empty state (no error).
* Malformed/unreadable file → `ConfigError`; the client reports it and starts
  with an empty state (optionally backing up the bad file).
* Unknown keys and a different `version` are tolerated; invalid connection entries
  are skipped.
* The file is written after every state change that matters (add/remove
  connection, login/register/logout/delete) via a single `_persist()` helper in
  the controller.

---

## 6. Handler parity (WebUI route → controller operation)

The desktop client has no URLs; the table maps each WebUI route to the equivalent
controller method and the api_server call it performs. (WebUI routes are in
[`../../docs/architecture.md`](../../docs/architecture.md) § 7.)

| WebUI route | api_server call | Controller method | Result in the client |
|---|---|---|---|
| `GET /` | — | `show_servers` | mount `ServersView` |
| `POST /servers` | — | `add_server` | persist, `show_server` |
| `GET /servers/{id}` | — | `show_server` | mount `ServerView` |
| `POST /servers/{id}/remove` | — | `remove_server` | persist, `show_servers` |
| `POST /servers/{id}/login` | `POST /user/login` | `log_in` | store session, persist, `show_server` |
| `POST /servers/{id}/register` | `POST /user/register` | `register` | store session, persist, `show_server` |
| `POST /servers/{id}/logout` | `POST /user/logout` | `log_out` | clear session, persist, `show_server` |
| `POST /servers/{id}/delete` | `POST /user/delete` | `delete_account` | clear session, persist, `show_server` + notice |
| `GET /servers/{id}/chats` | `GET /chat/get_chats` | `load_chats` | mount `ChatsView` |
| `POST /servers/{id}/chats` | `POST /chat/create` | `create_chat` | `show_chat(new_uid)` |
| `POST /servers/{id}/chats/join` | `POST /chat/join_chat` | `join_chat` | `show_chat(new_uid)` |
| `GET …/chats/{uid}` | `GET /chat/get_chats?uids=uid` + `GET …/get_messages` + `GET …/get_members` + `GET /user/get` | `load_chat` | mount `ChatView` with data |
| `POST …/messages` | `POST /chat/{uid}/send_message` | `send_message` | reload chat |
| `POST …/join_link` | `POST /chat/{uid}/create_join_link` | `create_join_link` | stash token, reload chat |
| `POST …/leave` | `POST /chat/{uid}/leave` | `leave_chat` | `show_chats` |

**Sign-in guard.** Chat routes in the WebUI redirect to the server page when
`connection.Session == nil`. The client does the same: every chat controller
method checks `connection.signed_in` first and navigates to `show_server`.

**One-shot join link.** `create_join_link` calls
`state.set_pending_join_link(server_id, chat_uid, token)` and then reloads the
chat; `load_chat` calls `state.take_pending_join_link(...)` and passes it to the
view. This preserves the WebUI's "show the token once, refresh is a no-op"
behaviour without HTTP redirects.

---

## 7. Error handling

| Source | Shape | Client behaviour |
|---|---|---|
| Backend validation (`400`/`422` with `field`) | `APIError(field, message)` | highlight the field, show `message` in the `Banner` |
| Backend auth failure (`401`) | `APIError(status=401)` | show the message and invite re-login (`show_server`) |
| Other backend failure | `APIError(status, message)` | show in the `Banner`; if it happens during a page load, `show_error` |
| Network failure | `APIError(status=None)` | "Could not reach <url>: …" in the `Banner` |
| Local validation | `ValidationError(field, message)` | shown immediately, before any request |
| Invalid/missing connection id | `ServerNotFoundError` | `show_error("server not found")` |
| Malformed config | `ConfigError` | dialog on startup, then an empty state |

Agents should follow the same rule as the Go handlers: **re-render the current
view with an error message** where a user can correct input, and use the error
view/dialog only for genuinely unexpected failures.

---

## 8. Navigation

`App.navigate(route)`:

1. destroy the current view frame (views are stateless, so building a page is
   cheap and forms are intentionally reset — like loading a fresh web page);
2. look up the view class for `route.name` in `App.VIEWS`;
3. instantiate, pack into the content area, store as `self._view`;
4. update the window title and status bar;
5. call `self._view.on_show()`, which asks the controller to load data.

`F5` / the **Refresh** menu item calls `controller.refresh()`, which re-runs the
current route's load (`load_chats`, `load_chat`, or nothing for
`servers`/`server`).

---

## 9. Conventions for implementers

* **Standard library only** — no `requests`, no `tkinter` add-ons.
* **Type hints everywhere**; `from __future__ import annotations` at the top.
* Keep the Go naming where it helps (`apiclient`, `ServerConnection`,
  `pending_join_link`) and use snake_case for Python.
* Every module keeps the docstring explaining which Go file it mirrors and which
  FR it serves.
* No `print` for user-visible errors — use the `Banner`, the error view or
  `dialogs`.
* Do not add files outside `gui_client/**`.
* Verification per task: `python3 -m compileall chat_client` and, for pure
  layers, the optional unit tests (task 11).

---

## 10. Future work (explicitly not in v1)

* Optional background polling for the open chat (would reuse `TaskRunner` +
  `widget.after`); deliberately excluded by requirement C5.
* Message pagination via `before_message_uid`/`after_message_uid`.
* Multiple local profiles / config switching.
* A platform keyring backend for tokens (`config` is the only module that would
  change).
* Markdown/emoji rendering and desktop notifications.
