# Task 05 — Background task runner

**Depends on:** 00. **Feeds into:** 06 (and every controller task).

## Goal

Guarantee the Tk main thread is never blocked by HTTP. Tkinter is single-threaded
and not thread-safe; `TaskRunner` runs work on a worker pool and delivers results
back to the main thread via a queue drained by a periodic `after()` callback.
This is the client's replacement for the Go server's "goroutine per request".

## Deliverables

| File | State |
|------|-------|
| `chat_client/tasks/runner.py` | **implement** |
| `chat_client/tasks/__init__.py` | done |

## Interface contract

```python
Work      = Callable[[], Any]
OnSuccess = Callable[[Any], None]
OnError   = Callable[[BaseException], None]
OnDone    = Callable[[], None]

class TaskRunner:
    def __init__(self, widget: tk.Misc, *,
                 max_workers: int = DEFAULT_WORKER_THREADS,
                 poll_interval_ms: int = TASK_POLL_INTERVAL_MS) -> None
    def submit(self, work, *, on_success=None, on_error=None, on_done=None) -> None
    def shutdown(self) -> None
```

## Required behaviour

1. `__init__` creates a `concurrent.futures.ThreadPoolExecutor(max_workers=...)`,
   a `queue.Queue` for results and a `threading.Event`/flag for shutdown, then
   schedules `self._widget.after(poll_interval_ms, self._poll)`.
2. `submit(work, ...)`: wrap the callbacks in a record, submit a closure to the
   executor that runs `work()`, catches `BaseException`, and pushes
   `(success, value_or_exc, callbacks)` onto the queue. **Nothing** is called on
   the worker thread except `work`.
3. `_poll` runs on the main thread: drain the queue (non-blocking), invoke
   `on_success(result)` **or** `on_error(exc)`, then `on_done()` for each item;
   reschedule itself with `after` unless shutting down.
4. A callback raising an exception must be caught and reported (use
   `tkinter.messagebox.showerror`) and must **not** stop the poll loop.
5. `shutdown()`: set the flag, `executor.shutdown(wait=False, cancel_futures=True)`,
   stop rescheduling. Idempotent.

## Design notes

* Deliver results through the queue + `after`, **not** by calling `after` from the
  worker thread (not guaranteed thread-safe in every Tk build).
* Do not expose futures to callers; the callback API keeps them on the main
  thread by construction.
* `on_error` receives the exception unchanged, so the controller can branch on
  `APIError`.
* Keep `max_workers` small (default `DEFAULT_WORKER_THREADS`, currently 4): the
  client makes few concurrent calls.

## Acceptance criteria

- [ ] With a `tk.Tk()` root, `submit(lambda: 1+1, on_success=record)` calls
      `record` with `2` on the main thread (assert `threading.current_thread()`
      is the main thread inside the callback).
- [ ] A raising `work` calls `on_error(exc)`; `on_done` still runs.
- [ ] A raising `on_success` is reported and later tasks still complete.
- [ ] `shutdown()` then `submit(...)` does not raise.
- [ ] Many quick submissions all deliver exactly once.
- [ ] `python3 -c "import chat_client.tasks"` passes without a display
      (module import must not create a Tk root).

## Out of scope

Progress reporting, cancellation of a single task, retries, `asyncio`.

## References

- `docs/architecture.md` § 3 (threading model).
