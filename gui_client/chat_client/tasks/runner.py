"""Background task execution with Tk-safe callbacks.

Tkinter is not thread-safe: widgets may only be touched from the thread running
``mainloop``. Network calls, however, must not block that thread or the window
freezes (the WebUI dodged this by being server-rendered; a desktop client cannot).

``TaskRunner`` bridges the two: work runs on a
:class:`concurrent.futures.ThreadPoolExecutor`; finished work is pushed onto a
:class:`queue.Queue`; a periodic ``widget.after`` poll drains the queue on the
main thread and invokes the caller's callbacks there. Callbacks therefore may
safely touch widgets and ``AppState``.

Usage::

    runner.submit(
        lambda: api.get_chats(base_url, token),
        on_success=lambda chats: view.show_chats(chats),
        on_error=lambda err: view.show_error(err),
    )
"""

from __future__ import annotations

import logging
import queue
import threading
import tkinter as tk
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from tkinter import messagebox
from typing import Any, Callable

from chat_client.constants import DEFAULT_WORKER_THREADS, TASK_POLL_INTERVAL_MS

logger = logging.getLogger(__name__)

#: Callback invoked in the thread pool: returns a value or raises.
Work = Callable[[], Any]
#: Callback invoked on the main thread with the returned value.
OnSuccess = Callable[[Any], None]
#: Callback invoked on the main thread with the raised exception.
OnError = Callable[[BaseException], None]
#: Callback invoked on the main thread after success/error, always.
OnDone = Callable[[], None]


@dataclass(frozen=True)
class _Callbacks:
    """The main-thread callbacks bundled with one submitted task."""

    on_success: OnSuccess | None
    on_error: OnError | None
    on_done: OnDone | None


#: Queue payload: ``(ok, value_or_exception, callbacks)``.
_Outcome = tuple[bool, Any, _Callbacks]


class TaskRunner:
    """Runs blocking work off the main thread, marshals results back to it.

    :param widget: any Tk widget; used only for its ``after`` scheduler, which
        ties polling to the widget's lifetime.
    """

    def __init__(
        self,
        widget: tk.Misc,
        *,
        max_workers: int = DEFAULT_WORKER_THREADS,
        poll_interval_ms: int = TASK_POLL_INTERVAL_MS,
    ) -> None:
        self._widget = widget
        self._poll_interval_ms = poll_interval_ms
        self._executor = ThreadPoolExecutor(
            max_workers=max_workers, thread_name_prefix="chat-task"
        )
        self._results: queue.Queue[_Outcome] = queue.Queue()
        self._shutdown = threading.Event()
        self._poll_handle: str | None = None
        self._schedule_poll()

    def _schedule_poll(self) -> None:
        """Arrange the next main-thread drain unless shutting down."""
        if self._shutdown.is_set():
            return
        self._poll_handle = self._widget.after(self._poll_interval_ms, self._poll)

    def submit(
        self,
        work: Work,
        *,
        on_success: OnSuccess | None = None,
        on_error: OnError | None = None,
        on_done: OnDone | None = None,
    ) -> None:
        """Queue ``work`` for a worker thread.

        On the main thread, in order: ``on_success(result)`` **or**
        ``on_error(exception)``, then ``on_done()``. Exceptions raised by the
        callbacks themselves must be reported (via :mod:`tkinter.messagebox`) and
        must not kill the poll loop.
        """
        if self._shutdown.is_set():
            return

        callbacks = _Callbacks(on_success, on_error, on_done)

        def run() -> None:
            # Runs on a worker thread: compute, then hand the outcome to the
            # main thread through the queue. Nothing else happens here.
            try:
                value = work()
            except BaseException as exc:  # noqa: BLE001 - forwarded to on_error
                self._results.put((False, exc, callbacks))
            else:
                self._results.put((True, value, callbacks))

        try:
            self._executor.submit(run)
        except RuntimeError:
            # The executor was shut down between the guard and submit(); swallow
            # the same way an explicit post-shutdown submit() is ignored.
            logger.debug("ignoring task submitted after shutdown")

    def _poll(self) -> None:
        """Drain finished work on the main thread and dispatch its callbacks."""
        self._poll_handle = None

        while True:
            try:
                ok, value, callbacks = self._results.get_nowait()
            except queue.Empty:
                break
            self._dispatch(ok, value, callbacks)

        # Rescheduling lives outside the drain loop so a callback that was
        # reported as broken still leaves the loop alive.
        self._schedule_poll()

    def _dispatch(self, ok: bool, value: Any, callbacks: _Callbacks) -> None:
        """Invoke one task's callbacks on the main thread, isolating failures."""
        try:
            if ok:
                if callbacks.on_success is not None:
                    callbacks.on_success(value)
            elif callbacks.on_error is not None:
                callbacks.on_error(value)
        except BaseException as exc:  # noqa: BLE001 - keep the poll loop alive
            self._report_callback_error(exc)
        finally:
            if callbacks.on_done is not None:
                try:
                    callbacks.on_done()
                except BaseException as exc:  # noqa: BLE001
                    self._report_callback_error(exc)

    def _report_callback_error(self, exc: BaseException) -> None:
        """Surface a broken callback without letting it unwind the poll loop."""
        logger.exception("unhandled exception in task callback", exc_info=exc)
        try:
            messagebox.showerror(
                "Chat",
                f"An unexpected error occurred:\n{exc}",
                parent=self._widget,
            )
        except Exception:  # noqa: BLE001 - dialog failure must not propagate
            logger.exception("failed to display the callback error dialog")

    def shutdown(self) -> None:
        """Cancel the poll loop and stop accepting work. Called on window close."""
        if self._shutdown.is_set():
            return
        self._shutdown.set()

        if self._poll_handle is not None:
            try:
                self._widget.after_cancel(self._poll_handle)
            except Exception:  # noqa: BLE001 - widget may already be destroyed
                logger.debug("could not cancel the pending poll callback", exc_info=True)
            self._poll_handle = None

        self._executor.shutdown(wait=False, cancel_futures=True)
