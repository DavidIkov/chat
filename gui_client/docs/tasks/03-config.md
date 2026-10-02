# Task 03 — Config store (atomic `0600` JSON)

**Depends on:** 00. **Feeds into:** 04, 06, 10.

## Goal

Persist the client's connections and tokens to a local JSON file so they survive
restarts (decision C3). This is the only module that touches the filesystem and
the only place that knows the on-disk schema, so a future keyring backend can
replace it without affecting anything else.

## Deliverables

| File | State |
|------|-------|
| `chat_client/config/store.py` | **implement** |
| `chat_client/config/__init__.py` | done |

## Interface contract

```python
def default_config_path() -> Path: ...

class ConfigStore:
    def __init__(self, path: Path | None = None) -> None
    @property? path: Path            # public attribute is fine
    def load(self) -> dict
    def save(self, data: dict) -> None
    def clear(self) -> None
```

## Schema (version 1)

```json
{
  "version": 1,
  "next_id": 2,
  "connections": [
    {"id": 1, "url": "http://localhost:8080", "name": "Local",
     "session": {"uid": 1, "token": "…"}},
    {"id": 2, "url": "https://chat.example.com", "name": "", "session": null}
  ]
}
```

## Behaviour

* `default_config_path()` — `$XDG_CONFIG_HOME/chat/gui_client.json`, else
  `~/.config/chat/gui_client.json`. Use `CONFIG_DIR_NAME` /
  `CONFIG_FILE_NAME` from `constants`. On Windows/macOS a later task can branch
  here; keep it one function.
* `load()`:
  * file missing → return the empty schema (`{"version": CONFIG_VERSION,
    "next_id": 0, "connections": []}`), **not** an error;
  * file present but unreadable/malformed JSON/top-level not an object →
    raise `ConfigError`.
* `save(data)`:
  * `mkdir(parents=True, exist_ok=True)` with mode `0o700` (`CONFIG_DIR_MODE`);
  * write JSON to a temp file **in the same directory** (so `os.replace` is
    atomic), `os.chmod(tmp, 0o600)`, `os.replace(tmp, self.path)`;
  * `data` is always stamped with `"version": CONFIG_VERSION`;
  * on failure raise `ConfigError`.
* `clear()` — remove the file if present (used by "forget everything").

## Steps

1. Implement `default_config_path` (honour `XDG_CONFIG_HOME`).
2. Implement `load` with the missing-file/malformed split.
3. Implement `save` with the atomic temp-file dance and restrictive modes.
4. Implement `clear`.

## Acceptance criteria

- [ ] `load()` on a missing path returns the empty schema without raising.
- [ ] `save({...})` then `load()` round-trips the connections.
- [ ] `stat(path).st_mode & 0o777 == 0o600` after `save`.
- [ ] A truncated/corrupt file raises `ConfigError` (not `JSONDecodeError`).
- [ ] `save` twice in a row leaves no `.tmp` files behind.
- [ ] No files written outside the config directory.

## Out of scope

Serialising `AppState` (task 04) — this module only moves raw dicts. Schema
migration beyond tolerating unknown versions.

## References

- `docs/requirements.md` decisions C1/C3 and NFR-3.
- `docs/architecture.md` § 5.
