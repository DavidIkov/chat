# chat

A minimal, self-hostable chat system written in Go. It ships two independent
servers:

- **`api_server`** — a JSON HTTP API with bearer-token authentication, backed by
  PostgreSQL. All reads are `GET`, all writes are `POST`.
- **`webui_server`** — a server-rendered HTML front end with **no JavaScript**.
  It is a proxy + renderer: the browser only ever talks to the WebUI server, and
  the WebUI server talks to one or more `api_server` instances on the user's
  behalf, keeping every bearer token server-side.

The WebUI is deliberately simple: plain HTML forms, server-side sessions in
memory, and the Post/Redirect/Get pattern for every mutation.

---

## Table of contents

- [Features](#features)
- [Architecture](#architecture)
- [Requirements](#requirements)
- [Quick start](#quick-start)
- [Configuration](#configuration)
- [Make targets](#make-targets)
- [Project structure](#project-structure)
- [HTTP API reference](#http-api-reference)
- [Database schema](#database-schema)
- [Development](#development)
- [Security notes](#security-notes)
- [License](#license)

---

## Features

- Register, log in, log out and delete an account on an `api_server`.
- Create chats, join a chat via a join token, leave a chat.
- Send and read messages (messages are only fetched when a page is loaded —
  no real-time updates).
- Generate join links with an optional lifetime and/or maximum number of uses.
- Add **multiple** `api_server` instances to one browser session and sign in to
  each with its own independent account and token.
- Works entirely without JavaScript.

---

## Architecture

```
        HTML forms / links                  JSON over HTTP
  ┌──────────┐   (no JS)        ┌───────────────┐   Bearer token   ┌──────────────┐
  │ Browser  │ ───────────────▶ │ webui_server  │ ───────────────▶ │ api_server A │
  │ (cookie  │ ◀─────────────── │   (proxy +    │ ◀─────────────── │  + Postgres  │
  │  only)   │   rendered HTML  │    renderer)  │                  └──────────────┘
  └──────────┘                  │               │   Bearer token   ┌──────────────┐
                                │  in-memory    │ ───────────────▶ │ api_server B │
                                │   sessions    │ ◀─────────────── │  + Postgres  │
                                └───────────────┘                  └──────────────┘
```

Key ideas:

- The browser **never** sees an `api_server` URL or token — it holds only an
  opaque `webui_session` cookie.
- The WebUI server stores, per browser session, a list of **server connections**,
  each with its own `api_server` token. This means different connections can be
  signed in as different users, even against the same `api_server`.
- All WebUI state is **in memory** and is lost when the WebUI server restarts.

---

## Requirements

- **Go 1.26+** (see `go.mod`; the servers use the `net/http` method/wildcard
  routing patterns from Go 1.22+).
- **PostgreSQL 16** for the `api_server`. The included
  `dev/docker-compose.yml` starts one for local development.
- **Docker + Docker Compose** (only if you use the bundled Postgres; you can use
  any reachable Postgres instead).
- A modern browser for the WebUI.

---

## Quick start

Run each step in its own terminal, from the repository root.

### 1. Start PostgreSQL

```sh
make pgup
```

This runs `docker compose up` inside `dev/` and starts Postgres on
`localhost:7337` with database `chatdb`, user `admin`, password `admin`.
To run it in the background instead:

```sh
cd dev && docker compose up -d && cd ..
```

### 2. Start the API server

```sh
make run-api
```

Equivalent to:

```sh
go run ./cmd/api_server \
  -listenURL :8080 \
  -dbURL "postgres://admin:admin@localhost:7337/chatdb?sslmode=disable"
```

The API server creates its tables on startup (if they do not exist) and then
listens on `:8080`.

### 3. Start the WebUI server

```sh
make run-webui
```

Equivalent to:

```sh
go run ./cmd/webui_server -listenURL :8000
```

Run this from the repository root, because the default template and static
directories are relative paths.

### 4. Open the app

Browse to **<http://localhost:8000>**, add the API server URL
(`http://localhost:8080`), then register or log in and start chatting.

### Stopping

```sh
make pgdown   # stop the Postgres container
```

Stop the two Go servers with `Ctrl+C` in their terminals.

---

## Configuration

### `api_server`

| Flag | Meaning | Required | Default |
|------|---------|----------|---------|
| `-listenURL` | Address to listen on (e.g. `:8080`) | yes | — |
| `-dbURL` | PostgreSQL connection string | yes | — |

### `webui_server`

| Flag | Meaning | Required | Default |
|------|---------|----------|---------|
| `-listenURL` | Address to listen on (e.g. `:8000`) | yes | — |
| `-templatesDir` | Directory containing the `*.html` templates | no | `internal/webui_server/templates` |
| `-staticDir` | Directory containing static assets (CSS) | no | `internal/webui_server/static` |

### Postgres

| Setting | Value |
|---------|-------|
| Host / port | `localhost:7337` (mapped to container port `5432`) |
| Database | `chatdb` |
| User / password | `admin` / `admin` |
| Connection string | `postgres://admin:admin@localhost:7337/chatdb?sslmode=disable` |
| Volume | `chat_pgdata` |
| Container | `chat_postgres` |

Both servers accept an `fs.FS` for templates and static assets, so switching to
`embed.FS` for a single-binary deployment needs no signature changes.

---

## Make targets

| Target | Description |
|--------|-------------|
| `make` / `make all` | Alias for `check`. |
| `make build` | Compile every package and binary (`go build ./...`). |
| `make vet` | Run `go vet ./...`. |
| `make fmt` | Format all Go source in place with `gofmt`. |
| `make fmt-check` | Fail if any Go file is not `gofmt`-clean. |
| `make check` | `fmt-check` + `vet` + `build`. |
| `make run-api` | Run the API server (needs Postgres). |
| `make run-webui` | Run the WebUI server. |
| `make pgup` | Start the local Postgres via Docker Compose (foreground). |
| `make pgdown` | Stop the local Postgres container. |
| `make clean` | Remove cached build artifacts. |

The Makefile exposes `GO`, `API_ADDR`, `WEBUI_ADDR` and `DB_URL` as overridable
variables, for example:

```sh
make run-api API_ADDR=:9000 DB_URL="postgres://user:pass@host:5432/db?sslmode=disable"
```

---

## Project structure

```
.
├── cmd/                              # Executable entry points (flag parsing + wiring)
│   ├── api_server/main.go            # API server: opens the DB, builds services/handlers, serves
│   └── webui_server/main.go          # WebUI server: loads templates/static, serves HTML
│
├── internal/
│   ├── api_server/                   # JSON HTTP backend
│   │   ├── handlers/                 # HTTP layer (parse → call service → write JSON)
│   │   │   ├── handlers.go           #   Handlers struct + CreateHandlers(mux, services)
│   │   │   ├── auth/                 #   Bearer-token middleware + request context
│   │   │   ├── middleware/           #   JSON decode/respond helpers, panic recovery
│   │   │   ├── users/                #   /user/* routes
│   │   │   └── chats/                #   /chat/* routes (+ membership middleware)
│   │   └── services/                 # Domain layer over the database
│   │       ├── services.go           #   Services struct + CreateServices(db)
│   │       ├── users/                #   UsersService (auth, sessions, CRUD)
│   │       └── chats/                #   ChatsService (chats, messages, members, join links)
│   │
│   ├── webui_server/                 # Server-rendered HTML front end
│   │   ├── handlers/                 # HTTP layer (parse form → call service → render)
│   │   │   ├── handlers.go           #   Handlers struct + CreateHandlers(...)
│   │   │   ├── chats/                #   Chat list / chat / message routes
│   │   │   ├── servers/              #   Landing page + per-connection routes
│   │   │   ├── middleware/           #   Session cookie middleware, panic recovery
│   │   │   ├── render/               #   Template loading + Render/Redirect helpers
│   │   │   └── routes/               #   Shared path-key constants ({server_id}, {chat_uid})
│   │   ├── services/                 # Domain layer
│   │   │   ├── services.go           #   Services struct + CreateServices()
│   │   │   ├── sessions/             #   In-memory WebUI sessions + server connections
│   │   │   └── apiclient/            #   Typed HTTP client for api_server
│   │   ├── templates/                # html/template files (layout + one file per page)
│   │   │   ├── layout.html           #   {{define "head"}} / {{define "foot"}} partials
│   │   │   ├── index.html            #   Server-connection list
│   │   │   ├── server.html           #   Login/register or signed-in view
│   │   │   ├── chats.html            #   Chat list + create/join forms
│   │   │   ├── chat.html             #   Messages + members + send/join-link/leave forms
│   │   │   └── error.html            #   500 error page
│   │   └── static/
│   │       └── style.css             # The single, small stylesheet
│   │
│   └── shared/                       # Contracts shared by both servers
│       ├── types.go                  #   shared.UID (uint32), shared.Time (Unix ms)
│       └── api/                      #   Request/response structs + validation
│           ├── error.go              #   api.Error{field, message}
│           ├── validator/            #   Validators + length limits
│           ├── user/                 #   User request/response types
│           └── chat/                 #   Chat/message/member request/response types
│
├── dev/
│   └── docker-compose.yml            # Local PostgreSQL for development
├── docs/
│   ├── requirements.md               # What the WebUI must do (decisions, FRs, NFRs)
│   └── architecture.md               # How the WebUI is built (packages, routes, models)
├── Makefile                          # Common developer tasks
├── go.mod / go.sum                   # Module `chat`
└── LICENSE                           # MIT
```

**Conventions.** Each domain package under `handlers/` and `services/` follows the
same file layout: `constructor.go` (wiring), `handlers.go`/`service.go` (logic),
`types.go` (structs) and `constants.go`/`errors.go` where needed. A top-level
`CreateX` function wires everything together, and methods use `this` as the
receiver name.

**Why `handlers/render` and `handlers/routes` are separate packages.** Both the
`servers` and `chats` handler packages need to render templates and both need the
shared path-key constants. Putting them in either would cause an import cycle, so
they live in small leaf packages that import nothing from `handlers`.

---

## HTTP API reference

All request and response bodies are JSON. Send the bearer token as
`Authorization: Bearer <token>`. Errors are returned as:

```json
{ "error": { "field": "name", "message": "..." } }
```

(`field` is optional and, when present, identifies the offending input.)

### Users

| Method | Path | Auth | Body / query | Response |
|--------|------|------|--------------|----------|
| `POST` | `/user/register` | — | `{name, password}` | `{user_session:{uid, token}}` |
| `POST` | `/user/login` | — | `{name, password}` | `{user_session:{uid, token}}` |
| `POST` | `/user/logout` | Bearer | — | `{}` |
| `POST` | `/user/delete` | Bearer | `{delete_messages}` | `{}` |
| `GET` | `/user/get` | Bearer | `?uids=1,2,3` | `{users:[{uid, name}]}` |

### Chats

| Method | Path | Auth | Body / query | Response |
|--------|------|------|--------------|----------|
| `POST` | `/chat/create` | Bearer | `{name}` | `{chat_uid}` |
| `GET` | `/chat/get_chats` | Bearer | `?uids=...` (**optional**) | `{chats:[{chat_uid, name, created_at, creator_user_uid}]}` |
| `GET` | `/chat/{uid}/get_messages` | Bearer, member | `?limit=&before_message_uid=&after_message_uid=` | `{messages:[{user_uid, chat_uid, message_uid, created_at, text}]}` |
| `POST` | `/chat/{uid}/send_message` | Bearer, member | `{text}` | `{message_uid}` |
| `GET` | `/chat/{uid}/get_members` | Bearer, member | — | `{members:[{user_uid}]}` |
| `POST` | `/chat/{uid}/create_join_link` | Bearer, member | `{lifetime_seconds, max_uses}` | `{token, expires_at}` |
| `POST` | `/chat/{uid}/leave` | Bearer | — | `{}` |
| `POST` | `/chat/join_chat` | Bearer | `{token}` | `{chat_uid}` |

> **`get_chats` with no `uids`** returns every chat the caller is a member of.
> When `uids` are supplied, only those chats are returned (the WebUI uses this to
> fetch a single chat's metadata).

### Validation limits

| Field | Constraint |
|-------|-----------|
| User name | 4–16 characters |
| User password | 4–16 characters |
| Chat name | 4–32 characters |
| Message text | 1–256 characters |
| `get_messages` `limit` | defaults to `50` when `0` |

### WebUI routes

For completeness, the browser-facing routes served by `webui_server`:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/` | List server connections / add a server |
| `POST` | `/servers` | Add a server connection |
| `GET` | `/servers/{server_id}` | Server page (login/register or signed-in) |
| `POST` | `/servers/{server_id}/remove` | Remove a connection |
| `POST` | `/servers/{server_id}/login` | Log in on a connection |
| `POST` | `/servers/{server_id}/register` | Register on a connection |
| `POST` | `/servers/{server_id}/logout` | Log out on a connection |
| `POST` | `/servers/{server_id}/delete` | Delete the account on a connection |
| `GET` | `/servers/{server_id}/chats` | Chat list |
| `POST` | `/servers/{server_id}/chats` | Create a chat |
| `POST` | `/servers/{server_id}/chats/join` | Join a chat via token |
| `GET` | `/servers/{server_id}/chats/{chat_uid}` | Chat page |
| `POST` | `/servers/{server_id}/chats/{chat_uid}/messages` | Send a message |
| `POST` | `/servers/{server_id}/chats/{chat_uid}/join_link` | Create a join link |
| `POST` | `/servers/{server_id}/chats/{chat_uid}/leave` | Leave a chat |
| `GET` | `/static/` | Static assets (CSS) |

---

## Database schema

Tables are created automatically by the `api_server` on startup
(`create table if not exists`), so no manual migration step is required:

- `users` — `uid` (serial PK), `name` (unique), `password_hash`.
- `chats` — `uid` (serial PK), `name`, `creator_user_uid` (FK `users`),
  `created_at`.
- `messages` — `uid` (serial PK), `user_uid` (FK `users`), `chat_uid` (FK
  `chats`), `created_at`, `text`.
- `chat_members` — `(chat_uid, user_uid)` composite PK.

---

## Development

```sh
make check   # gofmt check + go vet + go build
```

Other useful commands:

```sh
gofmt -l .        # list files that need formatting
go build ./...    # compile everything
go vet ./...      # static analysis
```

There are **no automated tests** in this project by design. Verification is done
with `go build`, `go vet`, `gofmt`, and manual smoke testing through the WebUI.

The full task breakdown lives in [`docs/requirements.md`](docs/requirements.md)
and [`docs/architecture.md`](docs/architecture.md).

---

## Security notes

- Bearer tokens never leave the WebUI server; the browser only holds an
  `HttpOnly`, `SameSite=Lax` session cookie.
- The WebUI validates user-supplied server URLs to `http`/`https`.
- User-supplied server URLs mean the WebUI will fetch whatever the user enters —
  run it locally or behind a network boundary / reverse proxy.
- The WebUI has no accounts of its own. Put it behind TLS and an allow-list
  before exposing it publicly.

---

## License

Released under the [MIT License](LICENSE).
