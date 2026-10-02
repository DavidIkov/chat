# GUI Client — Requirements

What the Python desktop client must do. The companion
[`architecture.md`](architecture.md) describes how it is built; the implementation
is split into tasks in [`tasks/`](tasks/README.md).

This is a **local desktop application**, not a server, so it satisfies the same
product requirements as `internal/webui_server` (see
[`../../docs/requirements.md`](../../docs/requirements.md)) with two deliberate
differences:

* **Persistence** — connections and tokens survive a restart (the WebUI is
  in-memory only).
* **Interactivity** — it is not restricted to "no JavaScript". It still refreshes
  **manually** (`FR-13`), but it may use real widgets, inline validation and a
  background worker pool.

Terminology:

- **api_server** — the Go backend under `internal/api_server` (JSON HTTP API,
  bearer tokens). Treated as read-only; this client never changes it.
- **connection** — one api_server instance the user added, with its own token.
- **client state** — the single `AppState` holding every connection.

---

## 1. Purpose

A native Tkinter front end over one or more `api_server` instances. The user adds
server URLs, signs in to each with that server's own account, and then reads and
writes chat messages using the api_server HTTP API. All state lives locally.

---

## 2. Decisions (agreed up front)

| # | Decision | Choice |
|---|----------|--------|
| C1 | Where tokens live | In a local config file, mode `0600`. It is the user's own machine, so no proxy is needed. |
| C2 | Backend traffic | The client talks **directly** to every api_server over JSON HTTP. No CORS, no intermediate server. |
| C3 | Persistence | Connections + per-connection sessions persist to `$XDG_CONFIG_HOME/chat/gui_client.json`. |
| C4 | Multi-server | Many api_servers at once, each with its **own independent** user session (parity with WebUI FR-14). |
| C5 | Interactivity | **Manual refresh** (`FR-13`). No polling, no websockets — the api_server has no push channel. |
| C6 | Front end | Tkinter/`ttk` widgets, one view per route, in-process router. |
| C7 | Contracts | The same JSON request/response shapes as `internal/shared/api/...`, modelled as dataclasses in `chat_client.models`. |
| C8 | Dependencies | **Standard library only** (`tkinter`, `urllib`, `json`, `concurrent.futures`). |
| C9 | Threading | Blocking HTTP runs on a worker pool; results return to the Tk main thread. The UI must never freeze. |
| C10 | Tests | Optional but encouraged for the pure layers (`models`, `apiclient`, `config`, `state`); no GUI tests. |

---

## 3. Functional requirements

### FR-1 — Add a connection
Add an api_server by **base URL**, with an optional display **name** (default: the
URL). The URL is validated locally (scheme `http`/`https`, non-empty host). On
success the client opens that connection's page.
* No api_server call at add time.*

### FR-2 — List connections
The landing page lists every connection: label, URL, whether a user is signed in,
plus "Open" and "Remove".

### FR-3 — Remove a connection
Remove a connection from state (and the persisted config). Clears its token
locally. Does **not** call the api_server.

### FR-4 — Sign in on a connection
Log in or register with that api_server's own name/password. On success the
returned `{uid, token}` is stored on the connection and the page reflects the
signed-in state.
* `POST /user/login`, `POST /user/register`.*

### FR-5 — Sign out of a connection
Log out on a connection: invalidate the token on the backend and drop it locally.
Other connections are unaffected.
* `POST /user/logout`.*

### FR-6 — List chats
When signed in, list every chat the account is a member of; each opens its chat
view.
* `GET /chat/get_chats` with **no `uids`** → all the caller's chats (backend
  decision D8).*

### FR-7 — Create a chat
Create a chat by name (4–32 chars); on success open the new chat.
* `POST /chat/create`.*

### FR-8 — Join a chat via token
Paste a join token to join a chat; on success open it. Invalid/expired/exhausted
tokens show the backend's message.
* `POST /chat/join_chat`.*

### FR-9 — Open a chat and view messages
Show the chat name, its most recent messages (author name + text) and its
members.
* `GET /chat/get_chats?uids={chat_uid}` (name)
* `GET /chat/{chat_uid}/get_messages`
* `GET /chat/{chat_uid}/get_members`
* `GET /user/get` (resolve uid → name; batch all member uids in one call).*

### FR-10 — Send a message
Post a message (1–256 chars); afterwards the chat reloads and the message is
visible.
* `POST /chat/{chat_uid}/send_message`.*

### FR-11 — Create a join link
Generate a join token, optionally with a lifetime and/or a maximum number of
uses. Both limits are **opt-in** (checkbox + value); when a limit is off, `0` is
sent meaning "unlimited/never expires". The token is displayed once, so a refresh
cannot re-mint it.
* `POST /chat/{chat_uid}/create_join_link`.*

### FR-12 — Leave a chat
Leave the open chat; afterwards return to the chat list.
* `POST /chat/{chat_uid}/leave`.*

### FR-13 — Manual refresh
New data is only fetched when the user refreshes. A **Refresh** action (button
and `F5`) reloads the current view's data, and a status-bar/footer hint states
that updates are not live.

### FR-14 — Multiple concurrent servers, isolated sessions
Signing in or out of one connection must never affect another, even for two
connections pointing at the same api_server.

### FR-15 — Delete the account
Permanently delete the signed-in account. A checkbox decides whether the account's
messages are also removed from every chat. Afterwards the connection is signed
out and a confirmation is shown.
* `POST /user/delete`.*

---

## 4. Non-functional requirements

### NFR-1 — Never block the UI
Every network call runs off the main thread; the window stays responsive and the
active view shows a busy state. Callbacks run on the main thread.

### NFR-2 — Standard library only
No third-party runtime dependencies. Tkinter for UI, `urllib` for HTTP, `json`
for serialisation, `concurrent.futures` for concurrency.

### NFR-3 — Safe persistence
The config file is written atomically with mode `0600` (directory `0700`). A
missing file is fine; a malformed file is reported and does not crash startup.

### NFR-4 — Reuse the backend contract
Request/response shapes match `internal/shared/api/...`. The client does not invent
new endpoints and does not modify the Go servers.

### NFR-5 — Parity of behaviour
For every WebUI route there is a single controller operation with the same effect
(authorisation checks, PRG-equivalent navigation, inline error re-render). See the
mapping table in `architecture.md` § 6.

### NFR-6 — Errors are handled, not fatal
Backend validation errors show next to the offending form; auth failures invite
re-login; unexpected failures show the error view or a dialog. No stack trace
reaches the user.

### NFR-7 — Portability
Runs on Linux, macOS and Windows where Tkinter is available. File paths go through
one function (`default_config_path`) so platform differences stay contained.

### NFR-8 — Concurrency correctness
`AppState` is only ever touched on the main thread; worker tasks receive plain
values. No shared mutable state crosses threads.

### NFR-9 — Readability and parity of naming
Module and method names mirror the Go packages (`apiclient`, `sessions`→`state`,
`handlers`→`controller`, `templates`→`ui/views`) so the two codebases can be read
side by side.

### NFR-10 — Tests
No GUI tests. Unit tests for the pure layers are optional (task 11) and must not
require a display.

---

## 5. Constraints

- Python 3.10+.
- Only `gui_client/**` is created. The Go servers and `internal/shared` are
  untouched.
- No new files outside `gui_client/`.

---

## 6. Out of scope

- Real-time updates (websockets/SSE/polling) — manual refresh only (C5).
- Any WebUI-of-its-own account system; the client has no accounts.
- Editing/deleting individual messages, avatars, read receipts, search.
- Rich message rendering (markdown), notifications, tray integration.
- Code signing / native installers.

---

## 7. Acceptance criteria (verified manually)

With a local `api_server` running and at least one other user/chat available:

1. Launching the client shows an empty connection list and the add-server form.
2. Adding the api_server lists it; opening it shows login/register.
3. Logging in (or registering) shows the signed-in uid and "Open chats".
4. The chat list shows chats (including ones created outside this client) and
   offers create/join.
5. Creating a chat opens it; a message can be sent and is visible after refresh.
6. A second api_server can be added and signed in as a different user; the two
   sessions remain independent.
7. Removing a connection returns to the landing page without it.
8. A join link can be created with no limits, and with a lifetime and/or a use cap
   enabled by its checkbox; the token is shown once.
9. Deleting the account (with and without "also delete my messages") signs the
   connection out and shows a confirmation.
10. Restarting the client restores the connections and their signed-in state from
    the config file.
11. While a slow/unreachable api_server is being contacted, the window stays
    responsive.
