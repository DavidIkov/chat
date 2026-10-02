# Chat — Desktop GUI Client

A native desktop client for the Chat `api_server`, written in Python with
**Tkinter** (standard library only, no third-party dependencies).

It does everything `internal/webui_server` does — add one or more `api_server`
instances, sign in to each independently, list/create/join chats, read and send
messages, see members, mint join links, leave chats and delete an account — but
as a local application instead of a server-rendered, no-JavaScript web page. So
the *no-JS limitation does not apply*: the client can use real widgets, inline
validation, a background worker pool and a native settings flow.

> **Status: complete.** Every module is implemented (no `NotImplementedError`
> remains) and mirrors the Go `webui_server` behaviour one page at a time. The
> per-feature implementation notes live in the tasks under
> [`docs/tasks/`](docs/tasks/README.md).

---

## How it relates to the Go servers

| Concern | `webui_server` (Go) | this client (Python) |
|---|---|---|
| Front end | `html/template`, one page per route | Tkinter frames, one view per route |
| Navigation | HTTP `303 See Other` (PRG) | in-process router (`Route` + `App.navigate`) |
| API access | server-side proxy keeps tokens | **direct** to `api_server`; tokens in a local `0600` file |
| Sessions | in-memory, keyed by cookie | one `AppState`, persisted to JSON |
| Multi-server | `WebUISession.Servers` | `AppState.connections` |
| Blocking I/O | goroutines per request | `ThreadPoolExecutor` + Tk `after` |
| Contracts | `internal/shared/api/...` structs | `chat_client.models` dataclasses |

The JSON contract is identical: see
[`internal/webui_server/services/apiclient`](../internal/webui_server/services/apiclient)
on the Go side and `chat_client/apiclient/paths.py` here.

---

## Requirements

- **Python 3.10+** with Tkinter available (`python3-tk` on Debian/Ubuntu; bundled
  with the python.org installers on macOS/Windows).
- A reachable `api_server` (see the repository [`README.md`](../README.md) to
  start Postgres + `api_server`).
- No `pip install` for runtime — standard library only.

## Running

From `gui_client/`:

```sh
python3 -m chat_client
```

with an explicit config file:

```sh
python3 -m chat_client --config ~/.config/chat/gui_client.json
```

Or install the console script (optional):

```sh
pip install --user -e .
chat-gui
```

The config file (server connections + bearer tokens) is stored at
`$XDG_CONFIG_HOME/chat/gui_client.json`, or
`~/.config/chat/gui_client.json` by default, with mode `0600`.

---

## Project structure

```
gui_client/
├── pyproject.toml                 # stdlib-only packaging + console script
├── README.md
├── tests/                         # headless unit tests for the pure layers
│   ├── test_models.py  test_state.py  test_config.py  test_apiclient.py
├── docs/
│   ├── requirements.md            # what the client must do (FRs/NFRs)
│   ├── architecture.md            # how it is built (layers, threading, routing)
│   └── tasks/                     # the implementation breakdown (start here)
│       ├── README.md              # task index + dependency graph
│       └── 00-…  11-….md         # one file per task
└── chat_client/                  # the application package
    ├── __main__.py                # entry point: `python -m chat_client`
    ├── constants.py               # limits, timeouts, UI + config constants
    ├── errors.py                  # AppError / APIError / ValidationError / …
    ├── models/                    # value objects (mirror internal/shared/api)
    │   ├── shared.py              #   Uid, TimestampMs, JSON coercion
    │   ├── user.py                #   User, UserSession
    │   └── chat.py                #   Chat, Message, Member, JoinLink
    ├── apiclient/                 # typed, stateless api_server client
    │   ├── paths.py               #   endpoint constants (mirror constants.go)
    │   ├── transport.py           #   JSON-over-HTTP + APIError normalisation
    │   ├── users.py               #   /user/* methods
    │   ├── chats.py               #   /chat/* methods
    │   └── client.py              #   ApiClient = Users + Chats + Transport
    ├── config/store.py            # atomic 0600 JSON config load/save
    ├── state/                     # in-memory state (mirror services/sessions)
    │   ├── session.py             #   AppState, ServerConnection, PendingJoinLink
    │   └── persistence.py         #   AppState <-> config JSON
    ├── tasks/runner.py            # worker pool + Tk-safe callback marshalling
    ├── controller/controller.py   # one method per user action (handler parity)
    └── ui/                        # the Tkinter layer
        ├── app.py                 #   Tk root, router, menu, status bar
        ├── route.py               #   Route / RouteName
        ├── dialogs.py             #   modal confirmations / message boxes
        ├── views/                 #   one BaseView per page
        │   ├── base.py  servers.py  server.py  chats.py  chat.py  error.py
        └── widgets/               #   Banner, ScrollableFrame, lists, form fields
            ├── banner.py  scrollable.py  lists.py  forms.py
```

---

## Development

There are no runtime dependencies; the pure layers (`models`, `apiclient`,
`config`, `state`) are covered by headless unit tests under `tests/`
(standard-library `unittest`, no third-party test runner).

```sh
python3 -c "import chat_client"          # import smoke check
python3 -m compileall chat_client        # syntax check
python3 -m unittest discover -s tests -v  # headless unit tests
python3 -m chat_client                   # run (needs a display)
```

See [`docs/requirements.md`](docs/requirements.md) for the functional
requirements and [`docs/architecture.md`](docs/architecture.md) for the design.
