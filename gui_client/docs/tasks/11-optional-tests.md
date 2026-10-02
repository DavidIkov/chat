# Task 11 — *(optional)* Unit tests for the pure layers

**Depends on:** 01, 02, 03, 04. **Independent of the GUI.**

## Goal

Add lightweight unit tests for the layers that have no Tk dependency, so the
contract-heavy code (JSON shapes, config schema, state transitions) is protected.
This is optional: the product decision for the Go WebUI was "no automated tests",
but a Python port with pure, side-effect-free modules makes cheap regression tests
worthwhile, and they run headlessly.

## Deliverables

```
gui_client/tests/
├── __init__.py
├── test_models.py       # from_dict coercion
├── test_state.py        # add/remove/session/pending-join-link + persistence
├── test_config.py       # atomic save, 0600, missing vs malformed
└── test_apiclient.py    # transport error mapping + query construction
```

Use the standard library `unittest` (no `pytest` dependency). Run with:

```sh
cd gui_client && python3 -m unittest discover -s tests -v
```

## What to cover

### `test_models.py`
* `from_dict` tolerates missing keys and returns defaults.
* Numeric/UID coercion (`"3"` → `int`), `None`/`""` → `0`.
* `UserSession.to_dict` round-trips through `from_dict`.

### `test_state.py`
* `validate_server_url`: accepts `http`/`https`, strips a trailing `/`; rejects
  `ftp://x` and `http://` (no host) with `InvalidServerUrlError`.
* `add_server` assigns increasing ids; `server_by_id` raises
  `ServerNotFoundError`; `remove_server` drops it.
* `set_server_session`/`clear_server_session`; `label` falls back to the URL.
* `take_pending_join_link` returns once, then `""`; a mismatch leaves it intact.
* `state_from_config(state_to_config(state))` preserves everything; `{}` and
  malformed entries do not raise.

### `test_config.py`
Use `tempfile.TemporaryDirectory` and an explicit `ConfigStore(path)`:
* missing file → empty schema, no raise;
* `save` → `load` round-trip; mode is `0o600`; parent dir created;
* corrupt JSON → `ConfigError`;
* no `.tmp` leftovers.

### `test_apiclient.py`
No real network: monkeypatch `urllib.request.urlopen` (or inject a fake opener)
and assert:
* `get_chats(..., uids=None)` builds no `uids` query; `[5]` builds `uids=5`;
* `get_chat_messages(..., limit=0)` omits `limit`; `limit=10` includes it;
* `get_users(..., [1,2])` sends the single comma-separated `uids=1,2`;
* a `{"error":{"field":"name","message":"too small user name"}}` body → `APIError`
  with `field`/`message`/`status`;
* a non-2xx plain-text body → `APIError` with the status and text;
* `URLError` → `APIError` with `status is None` and no exception leak;
* `Authorization: Bearer <token>` is set only when a token is given.

## Acceptance criteria

- [ ] `python3 -m unittest discover -s tests` passes with no display.
- [ ] Tests are deterministic and do not touch the real user config or the network.
- [ ] No third-party test dependencies.
- [ ] `tests/` is excluded from the package in `pyproject.toml` (or listed only as
      a dev extra).

## Out of scope

GUI tests, integration tests against a live `api_server`, mocking `TaskRunner`.

## References

- `docs/requirements.md` NFR-10 and decision C10.
- `internal/webui_server/services/apiclient/client.go` — the error-mapping contract
  to mirror in `test_apiclient.py`.
