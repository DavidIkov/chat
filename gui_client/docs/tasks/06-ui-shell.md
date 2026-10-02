# Task 06 — App shell, router, widgets & dialogs

**Depends on:** 00, 05. **Feeds into:** 07, 08, 09, 10.

## Goal

Build the window and the reusable UI furniture every page needs: the Tk root and
its router, the `BaseView` contract, the shared widgets, the modal dialogs and the
error page. This is the client's equivalent of `templates/layout.html` +
`handlers/render` + the static CSS.

Views for the three features (07–09) subclass `BaseView` and use the widgets from
this task, so their interfaces must match this brief exactly.

## Deliverables

| File | State | Notes |
|------|-------|-------|
| `chat_client/ui/app.py` | **implement** | Tk root, router, menu, status bar |
| `chat_client/controller/base.py` | **implement** | `ControllerBase`: navigation + helpers |
| `chat_client/ui/views/base.py` | **implement** | `BaseView` |
| `chat_client/ui/views/error.py` | **implement** | `ErrorView` |
| `chat_client/ui/widgets/banner.py` | **implement** | `Banner` |
| `chat_client/ui/widgets/scrollable.py` | **implement** | `ScrollableFrame` |
| `chat_client/ui/widgets/lists.py` | **implement** | `MessageList`, `MemberList` |
| `chat_client/ui/widgets/forms.py` | **implement** | `Labeled*` |
| `chat_client/ui/dialogs.py` | **implement** | modal helpers |
| `chat_client/ui/route.py` | done | `Route`, `RouteName` |

## 6.1 `App` (`ui/app.py`)

```python
class App(tk.Tk):
    def __init__(self, state: AppState, api: ApiClient, config: ConfigStore) -> None
    def navigate(self, route: Route) -> None
    def current_route(self) -> Route
    def show_banner_error(self, message: str) -> None
    def show_banner_notice(self, message: str) -> None
    def set_status(self, text: str) -> None
    def on_close(self) -> None
```

`__init__` responsibilities, in order:

1. `super().__init__()`; set `title(WINDOW_TITLE)`, `geometry(DEFAULT_WINDOW_SIZE)`,
   `minsize(*MIN_WINDOW_SIZE)`.
2. Build a `ttk` layout: a **menu bar** on the window, a **content frame**
   (`self._content`) that fills the window, and a **status bar** at the bottom.
3. Menu items:
   * *File* → **Add server** (`controller.show_servers`), **Refresh** (`F5`,
     `controller.refresh`), **Quit** (`on_close`);
   * *Help* → **About** (`dialogs.show_info`).
   Bind `<F5>` to `controller.refresh`.
4. `self.tasks = TaskRunner(self)` (task 05) — needs `self` as the widget.
5. `self.controller = Controller(self, state, api, self.tasks, config)`.
6. `self._view = None`, `self._route = None`;
   `self.protocol("WM_DELETE_WINDOW", self.on_close)`.
7. `self.controller.show_servers()`.

Routing:

```python
VIEWS = {
    RouteName.SERVERS: ServersView,
    RouteName.SERVER:  ServerView,
    RouteName.CHATS:   ChatsView,
    RouteName.CHAT:    ChatView,
    RouteName.ERROR:   ErrorView,
}
```

* `navigate(route)`: destroy `self._view` if present; instantiate `VIEWS[route.name]`
  with `(self._content, self, route)`; `pack(fill="both", expand=True)`; store;
  set `self._route = route`; update the title (append the connection label when
  the route names one); `self._view.on_show()`. Wrap the render in `try/except`
  and fall back to `Route.error(str(exc))` (guarding against infinite recursion by
  not re-entering on an error route).
* `current_route()` returns `self._route` (or `Route.servers()` before the first
  navigation).
* `show_banner_error/notice`: forward to `self._view.show_error/show_notice` when
  a view is mounted, else `dialogs.show_error`.
* `set_status(text)`: set the status-bar label to `text` (the default is
  `REFRESH_HINT`).
* `on_close()`: `self.tasks.shutdown()` then `self.destroy()`.

## 6.2 `BaseView` (`ui/views/base.py`)

```python
class BaseView(ttk.Frame):
    def __init__(self, master: tk.Misc, app: "App", route: Route) -> None
    def on_show(self) -> None: ...
    def show_error(self, message: str) -> None
    def show_notice(self, message: str) -> None
    def set_loading(self, loading: bool) -> None
    # provided for subclasses:
    self.app: App
    self.route: Route
    self.banner: Banner
    def build(self) -> None: ...   # optional subclass hook
```

`__init__` stores `app`/`route`, creates `self.banner = Banner(self)` packed at
the top, and calls `self.build()` so subclasses construct their widgets in a
single place. `show_error/show_notice` delegate to `self.banner`.
`set_loading(True)` disables interaction (e.g. sets a `state="disabled"` "busy"
flag) and shows an indeterminate `ttk.Progressbar`/label; `False` restores it.
Keep it simple: a `ttk.Label` with "Working…" plus disabling the view's buttons is
acceptable.

## 6.3 Widgets (`ui/widgets/`)

```python
class Banner(ttk.Frame):
    def show_error(self, message: str) -> None
    def show_notice(self, message: str) -> None
    def clear(self) -> None
    @property message -> str          # "" when hidden

class ScrollableFrame(ttk.Frame):
    def __init__(self, master, *, height: int = 240) -> None
    self.body: ttk.Frame              # add children here
    def refresh(self) -> None         # recompute scrollregion

class MessageList(ttk.Frame):
    def __init__(self, master, *, height: int = 240) -> None
    def set_messages(self, items: list[tuple[str, str]]) -> None   # (author, text)
    def show_empty(self, text: str = "No messages yet.") -> None

class MemberList(ttk.Frame):
    def __init__(self, master) -> None
    def set_members(self, names: list[str]) -> None

class LabeledEntry(ttk.Frame):
    def __init__(self, master, label, *, show=None, width=28, placeholder="") -> None
    def get(self) -> str;  def set(self, value: str) -> None;  def focus(self) -> None
class LabeledCheckbutton(ttk.Frame):
    def __init__(self, master, label, *, value=False) -> None
    @property var -> tk.BooleanVar;  def get(self) -> bool
class LabeledCombobox(ttk.Frame):
    def __init__(self, master, label, options: Sequence[tuple[str, object]]) -> None
    def get(self) -> object           # value behind the selected label
class LabeledSpinbox(ttk.Frame):
    def __init__(self, master, label, *, from_=1, to=1000, value=1) -> None
    def get(self) -> int
```

* `ScrollableFrame` = `Canvas` + inner frame + `ttk.Scrollbar`, with mouse-wheel
  binding (`<MouseWheel>` on X11/Windows, `<Button-4/5>` on some X11 setups) and
  re-`configure` of the scroll region on resize.
* Colours: crimson for errors, green for notices (mirror `static/style.css`).
* `MessageList` rows: bold author then the text, `wraplength` set so long messages
  wrap. `set_messages` clears and rebuilds; call `ScrollableFrame.refresh()`.

## 6.4 Dialogs (`ui/dialogs.py`)

```python
def show_info(parent, title, message) -> None
def show_error(parent, title, message) -> None
def confirm(parent, title, message) -> bool
def confirm_delete_account(parent, server_label: str) -> tuple[bool, bool]
```

* `show_info`/`show_error`/`confirm` wrap `tkinter.messagebox`.
* `confirm_delete_account` is a small custom `Toplevel` (transient + `grab_set`)
  with the warning text, a *"Also delete my messages from every chat"* checkbox
  and Delete/Cancel buttons; returns `(confirmed, delete_messages)`.

## 6.5 `ErrorView` (`ui/views/error.py`)

Renders `route.title`, `route.message`, and a **Back to servers** button
(`self.app.controller.show_servers()`). Port of `templates/error.html`.

## 6.6 `ControllerBase` (`controller/base.py`)

`App.__init__` constructs the `Controller`, and navigation goes through
`ControllerBase`, so this task also implements the shared controller
infrastructure. The domain mixins (`controller/servers.py`, `chats.py`, `chat.py`)
are implemented by tasks 07–09; this task only provides what they call.

```python
class ControllerBase:
    def __init__(self, app, state, api, tasks, config) -> None
    # navigation
    def show_servers(self) -> None
    def show_server(self, server_id: int, *, notice: str = "") -> None
    def show_chats(self, server_id: int) -> None
    def show_chat(self, server_id: int, chat_uid: int) -> None
    def show_error(self, message: str, *, title: str = "Error") -> None
    def refresh(self) -> None
    def current_route(self) -> Route
    # helpers used by the mixins
    def _persist(self) -> None
    def _server(self, server_id: int) -> ServerConnection
    def _signed_in_server(self, server_id: int) -> ServerConnection | None
    def _run(self, work, on_success, *, on_error=None, on_done=None) -> None
    def _show_api_error(self, error: BaseException) -> None
```

Behaviour:

* `__init__` stores `app`/`state`/`api`/`tasks`/`config`.
* Navigation methods call `self.app.navigate(Route...)`: `show_server` carries
  `notice` (for the post-delete banner — the WebUI's `?deleted=1`),
  `show_error` maps to `Route.error`.
* `refresh()` reads `self.current_route()` and re-triggers the matching load
  (`load_chats`, `load_chat`); for `servers`/`server` it simply re-navigates, and
  for `error` it does nothing.
* `_persist()` calls `self.config.save(state_to_config(self.state))`, converting a
  `ConfigError` into a non-fatal dialog.
* `_server` raises `ServerNotFoundError`; `_signed_in_server` returns `None` and
  navigates (error view when missing, `show_server` when signed out) so chat
  operations can bail without duplicating the guard.
* `_run(...)` delegates to `self.tasks.submit(...)` with `self._show_api_error` as
  the default error handler: an `APIError` goes to
  `self.app.show_banner_error(str(error))` (or the server page when signed out);
  any other exception goes to `show_error`.

## Acceptance criteria

- [ ] `App(state, api, config)` opens a window; when 07–09 are not yet done it is
      acceptable that navigating raises, but the window itself must appear and
      the menu/status bar must work.
- [ ] `navigate(Route.error("boom"))` shows `ErrorView` with "boom" and Back works.
- [ ] `navigate(Route.server(1))` calls the mount with the route intact (the
      `ServerView` body may still raise until task 07).
- [ ] `controller.show_*` and `controller.refresh()` route as documented above.
- [ ] `F5` and *File → Refresh* call `controller.refresh()`.
- [ ] Closing the window shuts the task runner down and exits cleanly.
- [ ] Each widget can be constructed in isolation inside a `tk.Tk()`; `Banner`
      hides itself when cleared; `MessageList.set_messages([])` shows the empty
      state.
- [ ] No third-party imports.

## Out of scope

The three feature views (07–09) and their controller methods. Theming beyond the
simple colour conventions.

## References

- `internal/webui_server/templates/layout.html`, `error.html`, `static/style.css`
- `internal/webui_server/handlers/render/render.go` (Render/Redirect analogue)
- `docs/architecture.md` § 4, § 8.
