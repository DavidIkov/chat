# Task 09 — Chat page feature

**Depends on:** 04 (state), 06 (shell/widgets/dialogs). **Feeds into:** 10.

## Goal

Implement the chat page: FR-9 (view messages + members), FR-10 (send a message),
FR-11 (create a join link, shown once) and FR-12 (leave the chat). Port of
`ChatPageHandler`, `SendMessageHandler`, `CreateJoinLinkHandler` and
`LeaveChatHandler` plus `templates/chat.html`.

## Deliverables

| File | Mirror | Implement |
|------|--------|-----------|
| `chat_client/controller/chat.py` | `handlers/chats/handlers.go` (chat page) | `ChatMixin` |
| `chat_client/ui/views/chat.py` | `templates/chat.html` | `ChatView`, `ChatMessageItem`, `ChatMemberItem` |

## 9.1 `ChatMixin` (controller)

### `load_chat(server_id, chat_uid)`

The most involved operation; it mirrors `renderChatPage` and performs up to four
api_server calls. Implement with a small sequence of `self._run(...)` steps (each
call may be chained from the previous success callback), or flatten with a single
worker function that performs all calls and returns a bundle — **the second is
recommended** for clarity:

```python
def work():
    chats = api.get_chats(url, token, [chat_uid])          # 1. name
    if not chats:
        raise ChatNotFound()                                # -> error view
    messages = api.get_chat_messages(url, token, chat_uid)  # 2. messages (no limit)
    members = api.get_chat_members(url, token, chat_uid)    # 3. members
    names = {}
    if members:                                             # 4. names (best effort)
        try:
            names = {u.uid: u.name for u in
                     api.get_users(url, token, [m.user_uid for m in members])}
        except APIError:
            names = {}
    return chats[0], messages, members, names
```

Then, on the main thread:

```python
join_link = state.take_pending_join_link(server_id, chat_uid)   # consume once
view.render_chat(chat, [ChatMessageItem(m, names.get(m.user_uid, "")) for m in messages],
                 [ChatMemberItem(m.user_uid, names.get(m.user_uid, "")) for m in members],
                 join_link)
```

Rules:

* Order and semantics match `renderChatPage` exactly: metadata first (empty result
  ⇒ "chat not found"), then messages, then members, then a **single batched**
  `get_users` whose failure is ignored (names fall back to `uid N`).
* `get_chats` here **does** pass `[chat_uid]` (unlike `load_chats`), which is how
  the WebUI resolves one chat's name.
* Guard with `self._signed_in_server(server_id)` first.
* A non-API failure during a load goes to the error view (`BadGateway` equivalent);
  a missing chat goes to `show_error("chat not found")`.

### `send_message(server_id, chat_uid, text)`

* Local validation: `MESSAGE_TEXT_MIN <= len(text) <= MESSAGE_TEXT_MAX` (1–256),
  else `ValidationError` and no request.
* `api.send_message(...)`, then on success `load_chat(server_id, chat_uid)` so the
  new message appears (the PRG equivalent). On failure show the banner and stay.

### `create_join_link(server_id, chat_uid, lifetime_seconds, max_uses)`

* `api.create_join_link(...)` with `0` meaning unlimited/never expires. The view
  is responsible for passing `0` when a checkbox is off.
* On success: `state.set_pending_join_link(server_id, chat_uid, link.token)` then
  `load_chat(server_id, chat_uid)`. `load_chat` consumes the token and the view
  shows it once — refreshing the page afterwards must **not** re-show it.
* On failure: banner (the "limit must be positive" messages are produced locally
  by the view before calling, so the backend error is usually auth/membership).

### `leave_chat(server_id, chat_uid)`

* `api.leave_chat(...)` then `show_chats(server_id)`. On failure, error view
  (the WebUI answers 502 here).

## 9.2 `ChatView` (`ui/views/chat.py`)

```python
class ChatMessageItem:  # message + resolved author name
    def __init__(self, message: Message, user_name: str = "") -> None
class ChatMemberItem:   # uid + resolved name
    def __init__(self, uid: int, name: str = "") -> None

class ChatView(BaseView):
    def on_show(self) -> None                            # controller.load_chat(...)
    def render_chat(self, chat, messages, members, join_link) -> None
    def set_join_link(self, token: str) -> None
```

`on_show` calls `self.app.controller.load_chat(self.route.server_id,
self.route.chat_uid)`. It performs no I/O itself.

Layout (top to bottom):

1. `Banner`.
2. Header "`chat.name`" with **← Chats** (→ `show_chats(id)`) and **Refresh**.
3. **Join link** strip: when `join_link` is non-empty, show
   "Join link token: `<token>`" (selectable/`code`-styled). Hidden otherwise.
   `render_chat` may draw it directly; `set_join_link` is provided for updates.
4. **Messages**: a `MessageList` fed with `(author_label, text)` pairs, where
   `author_label = item.user_name or f"uid {item.message.user_uid}"`. Empty →
   "No messages yet.".
5. **Send**: `LabeledEntry("Message")` (consider `maxlength` enforcement) + **Send**
   → `controller.send_message(id, chat_uid, text.get())`; clear the field after a
   successful send (rely on the reload).
6. **Members**: a `MemberList` of `member.name or f"uid {member.uid}"`.
7. **Share / create join link**:
   * `LabeledCheckbutton("Expire the link after a set time")` +
     `LabeledCombobox("Lifetime", JOIN_LINK_LIFETIME_OPTIONS)` (from `constants`);
   * `LabeledCheckbutton("Limit the number of uses")` +
     `LabeledSpinbox("Max uses", from_=1, value=JOIN_LINK_DEFAULT_MAX_USES)`;
   * **Create join link** → compute `lifetime = value if checkbox else 0`,
     `uses = value if checkbox else 0`, then
     `controller.create_join_link(id, chat_uid, lifetime, uses)`.
   * The dependent field should be disabled/hidden while its checkbox is off
     (the client is not limited by CSS/JS, so use `ttk` state).
   * If a checkbox is on but its value is invalid, show a local message and do not
     call the controller (parity with `parseJoinLinkLifetime`/`parseJoinLinkMaxUses`).
8. **Leave chat** button → `dialogs.confirm(...)` (recommended) then
   `controller.leave_chat(id, chat_uid)`.

## Acceptance criteria

- [ ] Opening a chat shows its name, messages (with author names where resolvable)
      and members.
- [ ] A chat whose metadata returns nothing shows the error view "chat not found".
- [ ] Sending a valid message reloads the chat and the message is visible; an empty
      or >256-char message is rejected locally.
- [ ] Messages/members with unresolvable uids show `uid N` instead of crashing.
- [ ] Creating a join link (unlimited) shows the token once; pressing Refresh does
      **not** show it again.
- [ ] Creating a join link with a lifetime and/or a use cap passes the chosen
      values; ticking a box with a non-positive value is rejected locally.
- [ ] Leaving returns to the chat list.
- [ ] The window stays responsive during every request; the view shows a busy state.

## Out of scope

Message pagination, editing/deleting messages, timestamps/relative time.

## References

- `internal/webui_server/handlers/chats/handlers.go`
  (`renderChatPage`, `ChatPageHandler`, `SendMessageHandler`,
  `CreateJoinLinkHandler`, `parseJoinLinkLifetime`, `parseJoinLinkMaxUses`,
  `LeaveChatHandler`)
- `internal/webui_server/templates/chat.html`
- `chat_client/constants.py` (`JOIN_LINK_*`, `MESSAGE_*`)
- `docs/architecture.md` § 6 (one-shot join link).
