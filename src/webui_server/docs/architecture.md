# WebUI Server — Architecture

Companion to [`requirements.md`](./requirements.md). This file explains how the
`webui_server` is structured and lays out the package/file skeleton so that
implementation agents can fill in the bodies.

---

## 1. Overview and system context

The WebUI server is a Go HTTP server that renders HTML for a human and, on that
human's behalf, calls one or more `api_server` instances.

```
        HTML forms / links                  JSON over HTTP
  ┌──────────┐   (no JS)        ┌───────────────┐   Bearer token   ┌──────────────┐
  │ Browser  │ ───────────────▶ │ webui_server  │ ───────────────▶ │ api_server A │
  │ (cookie  │ ◀─────────────── │   (this)      │ ◀─────────────── │              │
  │  only)   │   rendered HTML  │               │                  └──────────────┘
  └──────────┘                  │  in-memory    │   Bearer token   ┌──────────────┐
                                │  sessions     │ ───────────────▶ │ api_server B │
                                └───────────────┘ ◀─────────────── │              │
                                                                   └──────────────┘
```

Three rules follow from this picture:

1. The browser **never** talks to an api_server directly (no CORS, no tokens in
   the browser). D1 + D2 from the requirements.
2. The WebUI server is a **proxy + renderer**. It holds the api_server tokens and
   turns JSON into HTML.
3. State is **in-memory** and keyed by two ids: the WebUI `SessionID` (cookie)
   and, inside it, a `ServerID` for each added api_server.

---

## 2. Key decisions and their consequences

| Decision | Consequence in code |
|----------|---------------------|
| Server-side proxy | A dedicated `services/apiclient` package is the only thing that speaks to api_server. Handlers call it and render the result. |
| Server-side sessions | A `services/sessions` store maps cookie value → `*WebUISession`. Tokens live only here. |
| In-memory | A `map` guarded by `sync.RWMutex`. A `sessions` interface boundary keeps handlers unaware of the storage mechanism. |
| Many servers, isolated auth | `WebUISession.Servers []*ServerConnection`, each with its own `*user.UserSession`. |
| No real-time | Every mutating handler ends with a redirect (PRG); every read fetches fresh data on load. |
| HTML templates | `html/template`, one file per page, shared partials in `layout.html`. |
| Reuse `shared` contracts | `apiclient` encodes/decodes using `src/shared/api/...` types. |
| List all chats | Added a separate `GetUserChats` api_server service function; `get_chats` with no `uids` returns all the caller's chats (D8). |
| No tests | No `*_test.go` files; verify with build/vet/gofmt and manual smoke testing (D9). |

---

## 3. Request lifecycle

Every browser request passes through the same pipeline:

```
http.ServeMux
   │
   ├─ middleware.Recover        (panic → 500, one outer wrapper)
   │
   ├─ middleware.RequireSession (per-route)     reads/creates the WebUI session
   │        │                                   from the cookie, stores it in ctx
   │        ▼
   ├─ handler                 (servers/*, chats/*)
   │        │  reads session from ctx
   │        │  looks up the ServerConnection by ServerID
   │        │  calls services.API (apiclient) with that connection's token
   │        ▼
   └─ render.Render / render.Redirect   → HTML or 303 See Other
```

`RequireSession` is applied to **every** route (including `/`), because every
page needs a session to hold its server list. The api_server call is made with
`r.Context()`, so a browser disconnect cancels the outbound request.

### Post/Redirect/Get

Every route that mutates state (add/remove server, login, register, logout,
create chat, join chat, send message, create join link, leave chat) answers with
`303 See Other` on success and lets the browser re-issue a `GET`. This is why the
product works with **zero JavaScript**: reloading is always safe.

---

## 4. Package layout

Mirrors the `src/api_server` layout (`handlers/` + `services/`, per-domain
sub-packages, `constructor.go` for wiring, `middleware/` for cross-cutting code).

```
src/webui_server/
├── main.go                     # flags, wiring, ListenAndServe
├── docs/
│   ├── requirements.md
│   └── architecture.md
├── prompts/                    # numbered agent tasks (01..) + README.md
│
├── handlers/                   # HTTP layer: parse form → call service → render
│   ├── handlers.go             # Handlers struct + CreateHandlers(mux, services, templates, staticFS)
│   ├── routes/
│   │   └── constants.go        # ServerIDPathKey, ChatUIDPathKey
│   ├── render/
│   │   └── render.go           # Templates, LoadTemplates, Render, Redirect, Page
│   ├── middleware/
│   │   ├── constants.go        # SessionCookieName
│   │   ├── session.go          # RequireSession, SessionFromContext
│   │   └── recover.go          # Recover
│   ├── servers/                # /, /servers, /servers/{server_id}/*
│   │   ├── constants.go
│   │   ├── constructor.go
│   │   ├── handlers.go
│   │   └── types.go
│   └── chats/                  # /servers/{server_id}/chats/*
│       ├── constants.go
│       ├── constructor.go
│       ├── handlers.go
│       └── types.go
│
├── services/                   # domain layer
│   ├── services.go             # Services{Sessions, API} + CreateServices
│   ├── sessions/               # in-memory WebUI session + server connections
│   │   ├── constructor.go
│   │   ├── errors.go
│   │   ├── service.go
│   │   └── types.go
│   └── apiclient/              # typed HTTP client for api_server
│       ├── constants.go        # bearerPrefix + endpoint paths
│       ├── constructor.go
│       ├── errors.go           # APIError
│       ├── client.go           # shared request/response helper (do)
│       ├── users.go            # Register/LogIn/LogOut/DeleteUser/GetUsers
│       └── chats.go            # CreateChat/GetChats/GetChatMessages/...
│
├── templates/                  # html/template files (one per page + partials)
│   ├── layout.html             # {{define "head"}} / {{define "foot"}}
│   ├── index.html
│   ├── server.html
│   ├── chats.html
│   ├── chat.html
│   └── error.html
│
└── static/
    └── style.css
```

### Why `handlers/render` and `handlers/routes` are separate packages

- `servers` and `chats` handlers both need to render templates and both need the
  path-key constants. If either lived in `handlers/servers`, the other would have
  to import it, and `handlers` already imports both → import cycle. Small leaf
  packages (`render`, `routes`) avoid this, mirroring how `api_server` puts
  shared path keys in `handlers/chats/middleware`.
- `render` and `routes` import nothing from `handlers`, so they can be imported
  freely.

---

## 5. Domain model (in-memory)

Defined in `services/sessions/types.go`.

```
SessionsService
├── mu       sync.RWMutex
└── sessions map[SessionID]*WebUISession

WebUISession                      (one browser)
├── ID      SessionID             (random hex; goes into the cookie)
├── mu      sync.RWMutex
├── nextID  ServerID              (monotonic counter for ServerID)
└── Servers []*ServerConnection

ServerConnection                  (one api_server instance)
├── ID      ServerID              (stable within the session; used in URLs)
├── URL     string                (base URL, e.g. http://localhost:8080)
├── Name    string                (display label; defaults to URL)
└── Session *user.UserSession     (nil until signed in; {uid, token})
```

Notes:

- `ServerID` is a `uint32` and appears in webui URLs as `{server_id}`.
- The api_server `chat_uid` also appears in webui URLs as `{chat_uid}`; it is
  always interpreted **in the context of the enclosing `{server_id}`**.
- All access to `SessionsService.sessions` takes the service lock; all access to
  a `WebUISession.Servers` takes that session's lock. Mutating helpers live on
  `WebUISession` so callers cannot forget the lock.

### Session cookie

- Name: `webui_session` (`middleware.SessionCookieName`).
- Value: the `SessionID` (random, e.g. 16 bytes hex).
- Attributes: `HttpOnly`, `SameSite=Lax`, `Path=/`. `Secure` is set when the
  deployment is known to be HTTPS (future flag).

---

## 6. Services layer

### `services/sessions` — WebUI session & connection store

Responsibilities:

- `Create(ctx) (*WebUISession, error)` — new random session id.
- `Get(id) (*WebUISession, bool)` — lookup for the middleware.
- `Delete(id)` — drop a session (used on logout-everything / future cleanup).
- `(*WebUISession) AddServer(url, name) (*ServerConnection, error)` — validate the
  URL (scheme `http`/`https`, non-empty host), assign the next `ServerID`.
- `(*WebUISession) ListServers() []*ServerConnection` — a locked copy of the
  connections for rendering.
- `(*WebUISession) ServerByID(id) (*ServerConnection, bool)`.
- `(*WebUISession) RemoveServer(id)`.
- `(*WebUISession) SetServerSession(id, *user.UserSession) error`.
- `(*WebUISession) ClearServerSession(id) error`.

Returned errors: `ServerNotFoundError`, `InvalidServerURLError`.

### `services/apiclient` — typed client for api_server

One `Client` holds an `*http.Client`. Every method takes the target `baseURL` and,
when required, the `token` explicitly — the client is **stateless** with respect
to WebUI sessions. This keeps it trivially reusable and testable.

Shared helper (`client.go`):

```
do(ctx, method, url, token, requestBody, responseBody) error
```

It: marshals `requestBody` (if non-nil), sets `Content-Type: application/json`
and `Authorization: Bearer <token>` (when token != ""), sends the request,
decodes into `responseBody`, and converts a JSON `{"error":{...}}` body or a
non-2xx status into `*APIError{Status, Field, Message}`.

Endpoint constants live in `constants.go`, e.g. `userLoginPath = "/user/login"`.

When the WebUI wants every chat the signed-in user belongs to, it calls
`GetChats` with **no** uids; the api_server then routes to its `GetUserChats`
path (D8). Passing uids fetches just those chats (used to look up a single
chat's name on the chat page).

---

## 7. Routes (webui_server)

Go 1.22+ `net/http` method+wildcard patterns, registered in the `constructor.go`
of each handler package. `{server_id}` = `routes.ServerIDPathKey`,
`{chat_uid}` = `routes.ChatUIDPathKey`.

### `handlers/servers`

| Method | Pattern | Handler | Success result |
|--------|---------|---------|----------------|
| GET | `/{$}` | `IndexHandler` | render `index.html` (server list) |
| POST | `/servers` | `AddServerHandler` | 303 → `/servers/{server_id}` |
| GET | `/servers/{server_id}` | `ServerPageHandler` | render `server.html` |
| POST | `/servers/{server_id}/remove` | `RemoveServerHandler` | 303 → `/` |
| POST | `/servers/{server_id}/login` | `LogInHandler` | 303 → `/servers/{server_id}` |
| POST | `/servers/{server_id}/register` | `RegisterHandler` | 303 → `/servers/{server_id}` |
| POST | `/servers/{server_id}/logout` | `LogOutHandler` | 303 → `/servers/{server_id}` |

### `handlers/chats`

| Method | Pattern | Handler | Success result |
|--------|---------|---------|----------------|
| GET | `/servers/{server_id}/chats` | `ChatsPageHandler` | render `chats.html` |
| POST | `/servers/{server_id}/chats` | `CreateChatHandler` | 303 → chat page |
| POST | `/servers/{server_id}/chats/join` | `JoinChatHandler` | 303 → chat page |
| GET | `/servers/{server_id}/chats/{chat_uid}` | `ChatPageHandler` | render `chat.html` |
| POST | `/servers/{server_id}/chats/{chat_uid}/messages` | `SendMessageHandler` | 303 → chat page |
| POST | `/servers/{server_id}/chats/{chat_uid}/join_link` | `CreateJoinLinkHandler` | render chat page w/ token |
| POST | `/servers/{server_id}/chats/{chat_uid}/leave` | `LeaveChatHandler` | 303 → chats page |

### Static

| Method | Pattern | Handler |
|--------|---------|---------|
| GET | `/static/` | `http.FileServerFS(staticFS)` |

**Form field names** (POST body, `application/x-www-form-urlencoded`):

| Form | Route | Fields |
|------|-------|--------|
| Add server | `POST /servers` | `url`, `name` (optional) |
| Login | `POST /servers/{server_id}/login` | `name`, `password` |
| Register | `POST /servers/{server_id}/register` | `name`, `password` |
| Create chat | `POST /servers/{server_id}/chats` | `name` |
| Join chat | `POST /servers/{server_id}/chats/join` | `token` |
| Send message | `.../{chat_uid}/messages` | `text` |
| Join link | `.../{chat_uid}/join_link` | `lifetime_seconds`, `max_uses` |

**Redirect targets.** `CreateChatHandler` and `JoinChatHandler` redirect to the
chat page for the uid returned by the api_server. `SendMessageHandler` redirects
back to the same chat page. `LogInHandler`/`RegisterHandler` redirect back to the
server page.

### Handler responsibilities (uniform shape)

1. `r.ParseForm()`.
2. Load the WebUI session from context (`middleware.SessionFromContext`).
3. Resolve the `ServerConnection` (and, for chat routes, require
   `connection.Session != nil`; otherwise redirect to the server page to sign in).
4. Call the matching `apiclient` method with `connection.URL` and
   `connection.Session.Token`.
5. On success: redirect (PRG) or render.
6. On `*APIError`: re-render the current page with the message in the view model
   (HTTP status from the error, or 200 so the message is visible).
7. On other errors: render `error.html` with a 500.

---

## 8. api_server reference (what `apiclient` wraps)

All bodies are JSON. Errors come back as `{"error":{"field":"...","message":"..."}}`
(field optional). Auth is `Authorization: Bearer <token>`.

### Users

| api_server endpoint | Body / query | Response | Used by |
|---------------------|--------------|----------|---------|
| `POST /user/register` | `{name, password}` | `{user_session:{uid,token}}` | FR-4 |
| `POST /user/login` | `{name, password}` | `{user_session:{uid,token}}` | FR-4 |
| `POST /user/logout` | — (Bearer) | `{}` | FR-5 |
| `POST /user/delete` | `{delete_messages}` (Bearer) | `{}` | (out of scope) |
| `GET /user/get` | `?uids=..` (Bearer) | `{users:[{uid,name}]}` | FR-9 |

### Chats

| api_server endpoint | Body / query | Response | Used by |
|---------------------|--------------|----------|---------|
| `POST /chat/create` | `{name}` (Bearer) | `{chat_uid}` | FR-7 |
| `GET /chat/get_chats` | `?uids=..` (Bearer) — **optional**; when omitted, returns every chat the caller belongs to (D8) | `{chats:[{chat_uid,name,created_at,creator_user_uid}]}` | FR-6, FR-9 |
| `GET /chat/{uid}/get_messages` | `?limit=&before_message_uid=&after_message_uid=` (Bearer, member) | `{messages:[{user_uid,chat_uid,message_uid,created_at,text}]}` | FR-9 |
| `POST /chat/{uid}/send_message` | `{text}` (Bearer, member) | `{message_uid}` | FR-10 |
| `GET /chat/{uid}/get_members` | — (Bearer, member) | `{members:[{user_uid}]}` | FR-9 |
| `POST /chat/{uid}/create_join_link` | `{lifetime_seconds,max_uses}` (Bearer, member) | `{token,expires_at}` | FR-11 |
| `POST /chat/{uid}/leave` | — (Bearer) | `{}` | FR-12 |
| `POST /chat/join_chat` | `{token}` (Bearer) | `{chat_uid}` | FR-8 |

### Validator limits (mirror these in the UI where helpful)

- user name 4–16, password 4–16
- chat name 4–32
- message 1–256
- `get_messages` default limit is 50 when `limit` is 0.

The request/response structs are imported from `src/shared/api/user` and
`src/shared/api/chat`; `shared.UID`/`shared.Time` come from `src/shared`.

---

## 9. Templates and view models

`html/template`, files under `templates/`, loaded once at startup via
`render.LoadTemplates(fs)` using `template.ParseFS(templatesFS, "*.html")`.

**Partials, not block-overrides.** `layout.html` defines header/footer partials:

```
{{define "head"}} ... <link rel="stylesheet" href="/static/style.css"> ... {{end}}
{{define "foot"}} ... {{end}}
```

Every page file is a complete document that calls `{{template "head" .}}` and
`{{template "foot" .}}`. Pages are rendered by **file name** (e.g.
`Templates.Render(w, status, "index.html", data)`). This avoids the "multiple
`{{define "content"}}`" conflict that block-overrides hit when all files are
parsed together.

**Shared page data.** `render.Page{Title string}` is embedded in every view
model so `{{.Title}}` always works in `head`.

Proposed view models (in each handler package's `types.go`):

```
servers.IndexPage   { render.Page; Servers []*sessions.ServerConnection }
servers.ServerPage  { render.Page; Server *sessions.ServerConnection; SignedIn bool }

chats.ChatsPage     { render.Page; Server *sessions.ServerConnection; Chats []chatapi.Chat }
chats.ChatPage      { render.Page; Server *sessions.ServerConnection; Chat chatapi.Chat
                      Messages []MessageItem; Members []MemberItem; JoinLink string }
chats.MessageItem   { Message chatapi.Message; UserName string }
chats.MemberItem    { UID shared.UID; Name string }
```

Templates receive only data the handler fetched; they never call the API.

---

## 10. Errors

- `apiclient` normalises every backend failure into `*APIError`. A validation
  error from the api_server (`field` set) is shown next to the relevant form;
  auth failures (`401`) prompt the user to sign in again.
- Handlers re-render the current page with an error message rather than a bare
  status page, so the user can correct input.
- Unexpected errors render `error.html` with status 500.
- Panics are caught by `middleware.Recover` and turned into a 500 (same as
  `api_server`).

HTML forms can only send `GET`/`POST`, which matches the api_server (all writes
are POST, all reads are GET). No method overrides or hidden `_method` fields are
needed.

---

## 11. Configuration

Flags on `main` (mirroring `api_server`'s flag style):

| Flag | Meaning | Default |
|------|---------|---------|
| `-listenURL` | address the WebUI server listens on (required) | — |
| `-templatesDir` | directory of `*.html` templates | `src/webui_server/templates` |
| `-staticDir` | directory of static assets | `src/webui_server/static` |

`main` loads templates with `os.DirFS(*templatesDir)` and serves static files
with `http.FileServerFS(os.DirFS(*staticDir))`. Both accept an `fs.FS`, so a
later switch to `embed.FS` (for a single-binary deploy) needs no signature
changes.

---

## 12. Security considerations

- **Tokens never reach the browser** (D2). They are held in `ServerConnection`
  and only ever written into the `Authorization` header by `apiclient`.
- **Cookie**: `HttpOnly`, `SameSite=Lax`, `Path=/`; `Secure` to be added behind
  HTTPS.
- **CSRF**: `SameSite=Lax` blocks cross-site POSTs from other origins, which is
  sufficient for the intended minimal/local deployment. A CSRF token is noted as
  future work for exposed deployments.
- **SSRF**: the user supplies api_server URLs, so the WebUI will fetch them.
  Restrict schemes to `http`/`https`; treat "only allow hosts the user can reach"
  as an operational concern/future allow-list.
- **No authentication of the WebUI itself** in v1 — run it locally or behind a
  reverse proxy / network boundary.

---

## 13. Conventions

- Package layout, file naming (`constructor.go`, `handlers.go`, `types.go`,
  `constants.go`, `errors.go`), and `CreateX` wiring match `src/api_server`.
- Go receivers use `this` for service/handler methods, matching the existing
  code.
- No third-party dependencies; standard library only.
- Errors that are part of an API contract are exported `var XError = errors.New(...)`
  in the relevant package's `errors.go`.
- No tests in v1 (D9): do not add `*_test.go` files.
- Verification is `go build ./...`, `go vet ./...`, and `gofmt`.

---

## 14. Implementation checklist (for agents)

Work top-down; each step should compile on its own. The same sequence, as
ready-to-paste prompts, lives in `src/webui_server/prompts/` (see its
`README.md`).

0. **api_server (D8)** — add `GetUserChats` to `services/chats`, make the
   `get_chats` handler use it when `uids` is empty, and mark
   `GetChatsRequest.UIDs` as `form:"uids,omitempty"`. (Prompt 01.)

1. **`services/sessions`** — implement the store: random `SessionID`,
   `map` + mutex, `AddServer`/`ServerByID`/`RemoveServer`/`SetServerSession`/
   `ClearServerSession`, URL validation, `ServerNotFoundError`/`InvalidServerURLError`.
2. **`services/apiclient`** — implement `do` (JSON encode/decode, bearer header,
   `{error}` → `*APIError`, non-2xx → `*APIError`), then the `users.go` and
   `chats.go` methods using the `shared/api` types.
3. **`handlers/render`** — implement `LoadTemplates` (`template.ParseFS`),
   `Render` (execute by name, write status + content type), `Redirect` (303).
4. **`handlers/middleware`** — implement `RequireSession` (read cookie, look up
   or create session, set cookie, put session in context) and keep `Recover`.
5. **`handlers/servers`** — implement the seven handlers; wire routes in
   `constructor.go`; define view models in `types.go`.
6. **`handlers/chats`** — implement the seven handlers; wire routes; view models.
7. **`templates/`** — flesh out `layout.html` partials and each page using the
   view models; forms post to the routes above; add a footer refresh hint.
8. **`static/style.css`** — a small stylesheet (readable, not fancy).
9. **`main.go`** — already wired; adjust flag defaults if desired.
10. **Manual verification** — start the WebUI against a local api_server and
    walk the acceptance criteria in `requirements.md` with JS disabled.

---

## 15. Future work (explicitly not in v1)

- Real-time message delivery (SSE/WebSocket) — the requirement is manual reload.
- Optional persistence of connections/sessions.
- JS convenience layer (e.g. auto-scroll, in-place send) — progressive only.
- CSRF tokens, WebUI-level auth, HTTPS termination, SSRF allow-list.
- Rich message rendering (timestamps formatting, pagination via
  `before_message_uid`/`after_message_uid`).
- Account deletion (`POST /user/delete`) and message management.
