# Task 10 — Integration, polish & manual verification

**Depends on:** 07, 08, 09. **Final task (before the optional 11).**

## Goal

Turn the separately built layers into a coherent application: make sure the router
reaches every view, refresh and busy states behave, persistence is written on every
relevant change, and the whole acceptance checklist from
[`../requirements.md`](../requirements.md) § 7 passes against a real `api_server`.

## Deliverables

| File | Change |
|------|--------|
| `chat_client/ui/app.py` | wire `VIEWS`, titles, menu, refresh, close |
| `chat_client/controller/base.py` | `refresh()` dispatch, `_persist` error handling |
| `chat_client/controller/{servers,chats,chat}.py` | `_persist()` after every mutation; consistent error routing |
| `chat_client/ui/views/*.py` | busy states, field focus, empty/loading polish |
| `gui_client/README.md` | confirm the run instructions and structure match reality |

## Integration checklist

1. **Router coverage.** `App.VIEWS` maps every `RouteName` to a view;
   `Route.servers()/server()/chats()/chat()/error()` each mount the right one.
   Navigating through servers → server → chats → chat → back works and never
   leaves a stale frame packed.
2. **Refresh dispatch.** `controller.refresh()` re-runs the current route's load:
   `servers` (no-op / re-render), `server` (re-render), `chats` → `load_chats`,
   `chat` → `load_chat`, `error` (no-op). `F5` and *File → Refresh* both work.
3. **Persistence.** After every mutation (`add_server`, `remove_server`, `log_in`,
   `register`, `log_out`, `delete_account`) the config file is written. Restarting
   the client restores connections and signed-in state. A malformed config shows a
   warning and starts empty without crashing.
4. **Busy states.** Every view that loads data calls `set_loading(True)` before the
   request and `False` afterwards (success *and* failure paths — use `on_done`).
5. **Error routing.** A backend validation error appears in the current view's
   banner; an unreachable server gives "Could not reach …" and keeps the UI usable;
   a missing connection/chat shows the error view.
6. **Status bar / title.** The window title and status bar reflect the active
   connection (e.g. `Chat — Local`) and the `REFRESH_HINT`.
7. **No stray `NotImplementedError`.** Search the package:
   `grep -rn "NotImplementedError" chat_client` returns nothing.
8. **Keyboard/menu polish.** `F5` refresh, `Return` submits the focused form where
   it makes sense, initial focus lands in the first entry of each page, `Escape`
   closes dialogs.

## Manual acceptance walkthrough

Run a local stack (from the repository root):

```sh
make pgup        # Postgres
make run-api     # api_server on :8080
```

then, from `gui_client/`:

```sh
python3 -m chat_client
```

Walk every item in `docs/requirements.md` § 7 (acceptance criteria 1–11). Use two
`api_server` instances (e.g. second on `:8081` with its own database) for items 6
and 8, and a second client/`curl` to create a chat seen by the client in item 4.

Record the result of each item in the task's PR/commit description.

## Acceptance criteria

- [ ] All 11 acceptance items in `docs/requirements.md` § 7 pass.
- [ ] Restart persistence works (item 10) and a corrupt config recovers.
- [ ] `grep -rn "NotImplementedError" chat_client` is empty.
- [ ] `python3 -m compileall chat_client` and
      `python3 -c "import chat_client"` pass.
- [ ] No file outside `gui_client/**` was modified.
- [ ] `gui_client/README.md` instructions work verbatim.

## Out of scope

New features; real-time updates (explicitly excluded by decision C5).

## References

- `gui_client/docs/requirements.md` § 7
- `gui_client/docs/architecture.md` § 8, § 9
- Root `README.md` (running `api_server` + Postgres).
