# Task 01 — Domain models

**Depends on:** 00. **Feeds into:** 02, 04, 07, 08, 09.

## Goal

Model the api_server JSON contract as frozen dataclasses so the rest of the client
runs on typed objects instead of raw dicts. This is the Python counterpart of
`internal/shared/api/{user,chat}` plus `internal/shared/types.go`.

## Deliverables

| File | Mirror |
|------|--------|
| `chat_client/models/shared.py` | `internal/shared/types.go` |
| `chat_client/models/user.py` | `internal/shared/api/user/{types.go}` |
| `chat_client/models/chat.py` | `internal/shared/api/chat/{types.go}` |
| `chat_client/models/__init__.py` | re-exports |

## Interface contract

```python
# shared.py
Uid = int
TimestampMs = int
def uid_from_json(value) -> Uid: ...          # None/"" -> 0
def timestamp_from_json(value) -> TimestampMs: ...

# user.py
@dataclass(frozen=True, slots=True)
class User:        uid: int; name: str
    @classmethod from_dict(cls, data) -> "User"
@dataclass(frozen=True, slots=True)
class UserSession: uid: int; token: str
    @classmethod from_dict(cls, data) -> "UserSession"
    def to_dict(self) -> dict          # needed by config persistence (task 04)

# chat.py
@dataclass(frozen=True, slots=True)
class Chat:      chat_uid: int; name: str; created_at: int = 0; creator_user_uid: int = 0
@dataclass(frozen=True, slots=True)
class Message:   message_uid: int; text: str; user_uid: int = 0; chat_uid: int = 0; created_at: int = 0
@dataclass(frozen=True, slots=True)
class Member:    user_uid: int
@dataclass(frozen=True, slots=True)
class JoinLink:  token: str; expires_at: int = 0
# every class gets from_dict(cls, data)
```

## JSON key reference (must match exactly)

| Object | Keys |
|--------|------|
| `User` | `uid`, `name` |
| `UserSession` | `uid`, `token` |
| `Chat` | `chat_uid`, `name`, `created_at`, `creator_user_uid` |
| `Message` | `message_uid`, `text`, `user_uid`, `chat_uid`, `created_at` |
| `Member` | `user_uid` |
| `JoinLink` | `token`, `expires_at` |

The Go structs use `omitempty`, so **keys are often absent**; `from_dict` must
tolerate missing keys and fall back to the dataclass defaults.

## Steps

1. Implement the coercion helpers with the `None`/`""` → `0` rule.
2. Implement each `from_dict` using `data.get(key, default)`, coercing numbers
   with `int(...)` and strings with `str(...)`.
3. Add `UserSession.to_dict()` for the config schema.
4. Re-export everything from `models/__init__.py`.

## Acceptance criteria

- [ ] Instances are hashable and immutable (`frozen=True`).
- [ ] `Chat.from_dict({})` and `Message.from_dict({"text": "hi"})` do not raise.
- [ ] `UserSession.from_dict({"uid": "3", "token": "t"}).uid == 3`.
- [ ] `Message.from_dict` maps `chat_uid`/`message_uid`/`user_uid` to ints, not
      strings.
- [ ] `python3 -c "import chat_client.models"` passes.

## Out of scope

Request/response envelope objects (`{"error": ...}`, `{"user_session": ...}`) —
the transport layer (task 02) handles envelopes and returns these entities.

## References

- `internal/shared/types.go`, `internal/shared/api/user/types.go`,
  `internal/shared/api/chat/types.go`.
