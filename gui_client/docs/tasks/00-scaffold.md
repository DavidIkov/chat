# Task 00 — Project scaffold, conventions & bootstrap

**Depends on:** nothing. **Feeds into:** 01–11.

## Goal

Establish the `gui_client/` project: package layout, shared constants and errors,
the CLI entry point, packaging metadata, and the written conventions every later
task follows. The directory tree already exists with interface stubs; this task
verifies/installs it and wires the bootstrap so the app *starts* (views may still
raise `NotImplementedError` until 06–09 land, but importing and argument parsing
must work).

## Deliverables

| File | State | Notes |
|------|-------|-------|
| `gui_client/chat_client/__init__.py` | done | package docstring, `__version__` |
| `gui_client/chat_client/constants.py` | done | limits, timeouts, UI/config constants |
| `gui_client/chat_client/errors.py` | done | exception hierarchy |
| `gui_client/chat_client/__main__.py` | **implement** | `build_parser()`, `main()` |
| `gui_client/pyproject.toml` | done | stdlib-only; `chat-gui` console script |
| `gui_client/.gitignore` | done | Python ignores |
| `gui_client/README.md`, `gui_client/docs/**` | done | docs |

## Interface contract

```python
def build_parser() -> argparse.ArgumentParser: ...
def main(argv: list[str] | None = None) -> int: ...
```

Bootstrap order (the analogue of `cmd/webui_server/main.go`):

```python
def main(argv=None) -> int:
    args = build_parser().parse_args(argv)
    logging.basicConfig(level=logging.DEBUG if args.verbose else logging.INFO)

    config = ConfigStore(args.config)               # task 03
    try:
        data = config.load()
    except ConfigError as exc:                       # malformed file
        dialogs/show a warning; data = {}
    state = state_from_config(data)                  # task 04

    api = ApiClient()                                # task 02
    app = App(state, api, config)                    # task 06 (composition root)
    app.mainloop()
    return 0
```

* `--config PATH` — override the config file location.
* `--verbose` / `-v` — debug logging.
* Wrap the `App(...)` construction so a `tk.TclError` (no display) prints a clear
  message and returns exit code `1` instead of a traceback.

## Steps

1. Fill `build_parser`: `prog="chat-client"`, a `--config` option with
   `type=Path, default=None`, and a `-v/--verbose` flag.
2. Implement `main` in the order above. Import lazily inside `main` so
   `python3 -c "import chat_client.__main__"` works headlessly.
3. Catch `ConfigError` and `tk.TclError` with friendly messages.
4. Confirm `python3 -m chat_client --help` prints usage.

## Acceptance criteria

- [ ] `python3 -m compileall chat_client` passes.
- [ ] `python3 -c "import chat_client"` passes without a display.
- [ ] `python3 -m chat_client --help` prints `--config` and `--verbose` and exits 0.
- [ ] No third-party imports anywhere.
- [ ] Errors during startup never show a raw traceback.
- [ ] `pip install -e .` (optional) exposes `chat-gui`; the package list in
      `pyproject.toml` matches the real directories.

## Out of scope

Implementing any layer's logic; the UI (task 06). Other tasks may land before this
one if they only touch their own package.

## References

- `cmd/webui_server/main.go` — flag parsing and wiring order.
- `docs/requirements.md` § 2 (decisions C1, C8).
