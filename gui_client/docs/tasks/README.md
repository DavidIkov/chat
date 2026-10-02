# GUI Client — Implementation Tasks

The Python desktop client is built from ordered, mostly independent tasks. Each
file in this directory is a **self-contained brief** another agent can implement
without reading the others (though the interfaces are shared, so read §
"Shared interfaces" first).

Read [`../requirements.md`](../requirements.md) for *what* is required and
[`../architecture.md`](../architecture.md) for *how* the layers fit together.

---

## Task index

| # | Task | Owner layer | Depends on |
|---|------|-------------|-----------|
| [00](00-scaffold.md) | Project scaffold, conventions & bootstrap | package root | — |
| [01](01-models.md) | Domain models | `models/` | 00 |
| [02](02-apiclient.md) | api_server HTTP client | `apiclient/` | 00, 01 |
| [03](03-config.md) | Config store (atomic `0600` JSON) | `config/` | 00 |
| [04](04-state.md) | App state & persistence | `state/` | 00, 01, 03 |
| [05](05-task-runner.md) | Background task runner | `tasks/` | 00 |
| [06](06-ui-shell.md) | App shell, router, widgets, dialogs | `ui/` | 00, 05 |
| [07](07-servers.md) | Server connections feature | `controller/servers.py`, `ui/views/servers.py`, `ui/views/server.py` | 04, 06 |
| [08](08-chats-list.md) | Chat list feature | `controller/chats.py`, `ui/views/chats.py` | 04, 06 |
| [09](09-chat.md) | Chat page feature | `controller/chat.py`, `ui/views/chat.py` | 04, 06 |
| [10](10-integration.md) | Integration, polish & manual verification | all | 07, 08, 09 |
| [11](11-optional-tests.md) | *(optional)* unit tests for pure layers | `tests/` | 01, 02, 03, 04 |

## Dependency graph

```
00 scaffold
 ├── 01 models ─┬── 02 apiclient ─┐
 │             └── 04 state ──────┼───────────────┐
 │                 ▲             │               │
 ├── 03 config ────┘             │               │
 └── 05 task runner ── 06 ui shell┤               │
                                 ├── 07 servers ─┤
                                 ├── 08 chats list ┤── 10 integration
                                 └── 09 chat ─────┘        │
                                             11 (optional) ┘
```

**Suggested parallelisation.** After 00–06, tasks 07, 08 and 09 touch disjoint
files and can run in parallel. Task 10 must run last (it wires the router and
verifies everything end to end). Task 11 can run after 04 at any time.

---

## Shared interfaces (frozen — do not rename)

These names are already present as stubs; other tasks import them. Changing them
means changing several tasks at once.

| Symbol | Home | Purpose |
|---|---|---|
| `AppError`, `APIError`, `ValidationError`, `ConfigError`, `ServerNotFoundError`, `InvalidServerUrlError` | `errors.py` | exception hierarchy |
| `Uid`, `TimestampMs`; `User`, `UserSession`; `Chat`, `Message`, `Member`, `JoinLink` | `models/` | value objects with `from_dict` |
| `ApiClient.register/log_in/log_out/delete_user/get_users` | `apiclient/users.py` | account methods |
| `ApiClient.create_chat/get_chats/get_chat_messages/send_message/get_chat_members/create_join_link/join_chat/leave_chat` | `apiclient/chats.py` | chat methods |
| `ConfigStore.load/save/clear`, `default_config_path` | `config/store.py` | persistence |
| `AppState`, `ServerConnection`, `PendingJoinLink`, `validate_server_url`, `state_to_config`, `state_from_config` | `state/` | app state |
| `TaskRunner.submit/shutdown` | `tasks/runner.py` | concurrency |
| `Route`, `RouteName` | `ui/route.py` | navigation values |
| `App.navigate/current_route/show_banner_error/show_banner_notice/set_status` | `ui/app.py` | shell API |
| `BaseView.on_show/show_error/show_notice/set_loading` | `ui/views/base.py` | view contract |
| `Controller` + `*Mixin` methods | `controller/` | user actions |
| `Banner`, `ScrollableFrame`, `MessageList`, `MemberList`, `Labeled*` | `ui/widgets/` | reusable widgets |
| `dialogs.show_info/show_error/confirm/confirm_delete_account` | `ui/dialogs.py` | modal dialogs |

Constants (`chat_client.constants`) are the single source of truth for limits,
timeouts and presets.

---

## Definition of done (every task)

1. The listed files are fully implemented — no `NotImplementedError` remains in
   the modules that task owns.
2. Type hints on every public function; `from __future__ import annotations`.
3. `python3 -m compileall chat_client` passes.
4. `python3 -c "import chat_client"` passes.
5. The task's own "Acceptance criteria" checklist is satisfied.
6. Standard library only — no new third-party imports.
7. Only files inside `gui_client/**` are touched.
8. Module docstrings still name the Go file/FR they mirror.

## Agent conventions

* Read the referenced Go source under `internal/webui_server/**` when a behaviour
  is ambiguous — the Python client is a port of it.
* Never block the Tk main thread: any HTTP call goes through `TaskRunner`.
* Never touch `AppState` or widgets from a worker thread.
* Prefer mirroring the Go function names (`serverFromRequest` → `_server`,
  `SetPendingJoinLink` → `set_pending_join_link`) over inventing new vocabulary.
* If a shared interface must change, update the stub, this table, and the task
  files that reference it in the same change.
