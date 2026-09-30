# WebUI Server — Requirements

This document describes **what** the `webui_server` must do. The companion
document [`architecture.md`](./architecture.md) describes **how** it is built,
and [`../prompts/README.md`](../prompts/README.md) sequences the implementation
into ready-to-paste agent tasks.

Terminology used throughout:

- **WebUI server** — the new server implemented under `src/webui_server`. It is
  what the human opens in a browser.
- **api_server** — the existing backend under `src/api_server` (JSON HTTP API,
  bearer-token auth). The WebUI server calls it; it is treated as read-only except
  for the one approved change in D8.
- **Server connection** — one api_server instance that a user has added to the
  WebUI (identified by its base URL). A WebUI session can hold many.
- **WebUI session** — the server-side state tied to one browser (cookie), which
  owns the list of server connections and the credentials for each.

---

## 1. Purpose

Provide a **minimal, server-rendered web interface** on top of one or more
`api_server` instances. A user starts the WebUI server (locally or remotely),
opens it in a browser, adds one or more `api_server` URLs, signs in to each with
that server's own account, and then reads and writes chat messages using the
api_server HTTP API.

The emphasis is on **simplicity**: plain HTML forms, server-side rendering, and
no JavaScript. It is a thin, human-friendly shell over the existing API, not a
rich client.

---

## 2. Decisions (agreed up front)

| # | Decision | Choice |
|---|----------|--------|
| D1 | Browser ↔ backend traffic | **Server-side proxy.** The browser only ever talks to the WebUI server; the WebUI server talks to every api_server. No CORS, no backend URL/token exposure to JS. |
| D2 | Where bearer tokens live | **Server-side.** The browser holds only a WebUI session cookie; api_server tokens are kept in the WebUI server's memory. |
| D3 | Persistence | **In-memory only.** Server connections and sessions are lost when the WebUI server restarts. |
| D4 | State model | **Multiple api_servers at once**, each with its **own independent user session**. |
| D5 | Interactivity | **No real-time.** The user reloads the page to see new data. |
| D6 | Front end | **HTML templates rendered on the server**, no JS at all, minimal CSS in a static stylesheet. |
| D7 | Contracts | Reuse the Go structs in `src/shared/api/...` for requests/responses instead of re-declaring them. |
| D8 | Listing a user's chats | `GET /chat/get_chats` only filters by explicit `uids`, so it cannot list "all my chats" on its own. **Add a separate api_server service function `GetUserChats`** returning every chat the caller is a member of, and have the `get_chats` handler use it when no `uids` are provided. The `uids` field is marked `form:"uids,omitempty"`. This is the single approved change to `src/api_server`/`src/shared`. |
| D9 | Tests | **No automated tests** in the initial version. Verification is by `go build`, `go vet`, `gofmt`, and manual smoke testing. |

---

## 3. Functional requirements

Each requirement lists the user-visible behaviour and the api_server call(s) it
relies on (see the route table in `architecture.md` for the exact mapping).

### FR-1 — Add a server connection
The user can add an api_server by entering its **base URL** (e.g.
`http://localhost:8080`). Optionally a display **name** may be given; if omitted,
the URL is used as the label. The connection is added to the current WebUI
session and the user is taken to that server's page.
- No api_server call at add time (validation is local: scheme must be `http` or
  `https`).

### FR-2 — List server connections
The landing page lists every server connection in the session, showing its name
and URL, whether a user is signed in on it, and links to open it or remove it.
- No api_server call.

### FR-3 — Remove a server connection
The user can remove a connection from the session. Removal drops the local
connection and its stored credentials. It does **not** call the api_server
(nothing on the backend is deleted).
- No api_server call.

### FR-4 — Sign in on a server
For a given server connection, the user can **log in** or **register** using that
api_server's own name/password. On success the returned session (uid + token) is
stored on the connection and the server page reflects the signed-in state.
- `POST /user/login`
- `POST /user/register`

### FR-5 — Sign out of a server
The user can log out on a given server connection. The api_server token is
invalidated on the backend and dropped locally. Other connections are
unaffected.
- `POST /user/logout`

### FR-6 — List chats
When signed in, the user can view the list of chats that account is a member of
on that server. Each chat links to its chat page.
- `GET /chat/get_chats` with **no `uids`** → every chat the caller belongs to
  (requires the D8 change).
- `GET /chat/get_chats?uids=...` → only those chats; used to fetch a single
  chat's metadata (name) on the chat page.

### FR-7 — Create a chat
The user can create a chat by name; on success the user is taken to the new
chat's page. Chat name constraints are those of the api_server validator
(4–32 chars).
- `POST /chat/create`

### FR-8 — Join a chat via link
The user can paste a join token to join an existing chat; on success the user is
taken to that chat's page. Errors (invalid/expired/exhausted token) are shown to
the user.
- `POST /chat/join_chat`

### FR-9 — Open a chat and view messages
The chat page shows the chat's name, the most recent messages (author name and
text) and the chat's members. Message text limits follow the api_server validator
(1–256 chars).
- `GET /chat/get_chats?uids={chat_uid}` (chat metadata: name)
- `GET /chat/{chat_uid}/get_messages`
- `GET /chat/{chat_uid}/get_members`
- `GET /user/get` (to resolve member uid → display name)

### FR-10 — Send a message
The user can post a message to the open chat; after posting, the page reloads and
the new message is visible.
- `POST /chat/{chat_uid}/send_message`

### FR-11 — Create a join link
From a chat the user can generate a join link/token, optionally with a lifetime
and a maximum number of uses. Both limits are **opt-in**: a lifetime and a use
cap are only applied when the user ticks the matching checkbox, so the form never
asks the user to interpret a `0` default. The resulting token is displayed so it
can be shared: the request redirects back to the chat page (Post/Redirect/Get)
and the token is shown once there, so refreshing the page does not resubmit the
form.
- `POST /chat/{chat_uid}/create_join_link`

### FR-12 — Leave a chat
The user can leave the open chat; afterwards they are returned to the chat list.
- `POST /chat/{chat_uid}/leave`

### FR-13 — Manual refresh
New data is only fetched when a page is (re)loaded. The interface must make it
obvious that reloading is how you refresh (e.g. a short hint in the footer).

### FR-14 — Multiple concurrent servers, isolated sessions
Every server connection keeps its **own** api_server user session. Signing in or
out of one connection must never affect another, even if two connections point at
the same api_server.

### FR-15 — Delete the account
When signed in on a connection, the user can permanently delete that account. A
checkbox controls whether the account's messages are also removed from every
chat. Afterwards the connection is signed out (its stored token is invalid) and
the page shows a confirmation.
- `POST /user/delete`

---

## 4. Non-functional requirements

### NFR-1 — No JavaScript
Pages must work without JavaScript. The initial version ships **no JS**; a
progressive JS layer may be considered later but must never be required.

### NFR-2 — Server-rendered HTML, minimal CSS
Use Go `html/template`. Follow the Post/Redirect/Get pattern for all mutating
forms. CSS is a single small stylesheet.

### NFR-3 — In-memory state
All WebUI state (sessions, server connections, tokens) lives in memory behind a
small interface so a persistent store can be added later without touching
handlers.

### NFR-4 — Reuse shared contracts
Request/response bodies for api_server calls use the structs in
`src/shared/api/...` and `src/shared`.

### NFR-5 — api_server is read-only (with one exception)
`src/api_server` and `src/shared` must remain untouched, **except** for the single
approved change in D8 (the `GetUserChats` function and the `get_chats` no-`uids`
behaviour). No other edits to the existing server are allowed.

### NFR-6 — Style parity
Follow the conventions of `src/api_server`: `handlers/` + `services/`
sub-packages, `constructor.go`/`handlers.go`/`types.go`/`constants.go` file
layout, `middleware/` for cross-cutting concerns, a top-level `CreateX` wiring
function, and `this` as the receiver name.

### NFR-7 — Portability
Runs locally or on a remote host. Listen address and asset directories are
configurable via flags. No hard dependency on TLS (expected to sit behind a
reverse proxy when exposed).

### NFR-8 — Security posture
Bearer tokens never leave the WebUI server. The session cookie is `HttpOnly` and
`SameSite=Lax`. User-supplied server URLs are validated to `http`/`https`.

### NFR-9 — Correctness under concurrency
The in-memory store is safe for concurrent use (multiple browser tabs / users).

### NFR-10 — No automated tests
The initial version ships **no tests**. Do not create `*_test.go` files.
Verification is by `go build ./...`, `go vet ./...`, `gofmt`, and the manual
acceptance checks below.

---

## 5. Constraints

- Language/module: Go, module `chat` (see `go.mod`). Standard library only; no new
  third-party dependencies.
- Only `src/webui_server/**` (including `docs/` and `prompts/`) is created by the
  implementation. The sole exception is the D8 change in
  `src/api_server/services/chats`, `src/api_server/handlers/chats`, and
  `src/shared/api/chat`.
- No tests (NFR-10).

---

## 6. Out of scope (for the initial version)

- Real-time updates (websockets/SSE/polling).
- Rich client-side UI, SPA frameworks, bundlers, and JavaScript of any kind.
- Persistent storage across restarts.
- **Automated tests.**
- Password storage/synchronisation inside the WebUI (it delegates to api_server).
- Editing messages, deleting individual messages, avatars, read receipts, search.
- Multi-user WebUI accounts (the WebUI has no accounts of its own).

---

## 7. Acceptance criteria (verified manually)

A runnable demonstration is accepted when, with a local api_server running:

1. Starting the WebUI server and opening `/` shows an empty server list and an
   "add server" form.
2. Adding the api_server URL lists it; opening it shows login/register forms.
3. Logging in (or registering) shows the signed-in user and a link to chats.
4. The chats page lists chats (including ones created outside the WebUI) and
   offers create/join forms.
5. Creating a chat navigates to its page; a message can be posted and is visible
   after the reload.
6. A second, different api_server can be added and signed in as a different
   user; the two sessions remain independent.
7. Removing a connection returns to `/` without it.
8. A join link can be created with no limits, and with a lifetime and/or a use
   cap enabled through its checkbox.
9. Deleting the account (with and without "also delete my messages") signs the
   connection out and shows a confirmation.
10. With JavaScript disabled, all of the above works.

---

## 8. Glossary

| Term | Meaning |
|------|---------|
| WebUI session | Server-side record keyed by the browser's cookie. |
| Server connection | One api_server instance added to a WebUI session, with its own token. |
| PRG | Post/Redirect/Get — redirect after a successful POST so reloads don't resubmit. |
| Join link | A token created by `create_join_link` that lets another user join a chat. |
