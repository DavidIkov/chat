"""Chat-list operations — parity with the list/create/join parts of
``internal/webui_server/handlers/chats``.

Covers FR-6, FR-7 and FR-8. The chat *page* operations live in
:mod:`chat_client.controller.chat`. Mixed into
:class:`~chat_client.controller.controller.Controller`.

The three methods mirror one WebUI handler each:

* :meth:`load_chats` ← ``ChatsPageHandler`` (``GET /servers/{id}/chats``);
* :meth:`create_chat` ← ``CreateChatHandler`` (``POST /servers/{id}/chats``);
* :meth:`join_chat` ← ``JoinChatHandler`` (``POST /servers/{id}/chats/join``).

All of them resolve the connection with the shared sign-in guard first, hand the
blocking api call to :class:`~chat_client.tasks.runner.TaskRunner`, and then
either navigate to the affected chat (the WebUI's ``303`` redirect) or re-render
the chats page with the backend's message (``chats.html``'s ``Error`` field).
"""

from __future__ import annotations

from typing import Any

from chat_client.constants import CHAT_NAME_MAX, CHAT_NAME_MIN
from chat_client.errors import ValidationError
from chat_client.models.chat import Chat


class ChatsMixin:
    """``…/chats`` list, create and join operations."""

    # --- helpers -----------------------------------------------------------

    def _chats_view(self) -> Any:
        """Return the currently mounted view, or ``None``.

        The controller talks to the view through this instead of importing the
        ``ChatsView`` class, so the two layers stay independent. It also lets an
        operation ignore a result that arrives after the user navigated away.
        """
        return getattr(self.app, "_view", None)

    def _set_chats_loading(self, view: Any, loading: bool) -> None:
        """Toggle ``view``'s busy indicator, but only while it is still mounted."""
        if view is None or self._chats_view() is not view:
            return
        setter = getattr(view, "set_loading", None)
        if setter is not None:
            setter(loading)

    def _render_chats(self, view: Any, chats: list[Chat]) -> None:
        """Hand ``chats`` to the mounted chats view, when it is still current."""
        if view is None or self._chats_view() is not view:
            return
        render = getattr(view, "render_chats", None)
        if render is not None:
            render(chats)

    # --- operations --------------------------------------------------------

    def load_chats(self, server_id: int) -> None:
        """Fetch the caller's chats and render them (FR-6).

        ``GET /chat/get_chats`` is called with **no** uids, so the api_server
        returns every chat the caller belongs to (decision D8). On success the
        mounted view renders the list; on failure the WebUI re-renders
        ``chats.html`` with its ``Error`` field set, which the client mirrors by
        showing the message in the view's banner (falling back to the error view
        for a genuinely unexpected failure).
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        view = self._chats_view()
        self._set_chats_loading(view, True)
        self._run(
            lambda: self.api.get_chats(connection.url, session.token),
            lambda chats: self._render_chats(view, chats),
            on_done=lambda: self._set_chats_loading(view, False),
        )

    def create_chat(self, server_id: int, name: str) -> None:
        """``POST /chat/create`` then open the new chat (FR-7).

        The name is validated locally first (4–32 characters, see
        ``constants.CHAT_NAME_MIN/MAX``) so an obviously invalid value never
        reaches the backend. A local rejection, like a backend one, keeps the
        chats page on screen and reports the message in its banner.
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        name = name.strip()
        try:
            if len(name) < CHAT_NAME_MIN:
                raise ValidationError(
                    f"a chat name needs at least {CHAT_NAME_MIN} characters",
                    field="name",
                )
            if len(name) > CHAT_NAME_MAX:
                raise ValidationError(
                    f"a chat name can have at most {CHAT_NAME_MAX} characters",
                    field="name",
                )
        except ValidationError as exc:
            self.app.show_banner_error(str(exc))
            return
        view = self._chats_view()
        self._set_chats_loading(view, True)
        self._run(
            lambda: self.api.create_chat(connection.url, session.token, name),
            lambda chat_uid: self.show_chat(server_id, chat_uid),
            on_done=lambda: self._set_chats_loading(view, False),
        )

    def join_chat(self, server_id: int, token: str) -> None:
        """``POST /chat/join_chat`` then open the joined chat (FR-8).

        The token is only checked locally for emptiness; the backend's message
        (invalid/expired/exhausted token) is shown in the chats page banner and
        the user stays there to try another one.
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        token = token.strip()
        if not token:
            self.app.show_banner_error(
                str(ValidationError("a join token is required", field="token"))
            )
            return
        view = self._chats_view()
        self._set_chats_loading(view, True)
        self._run(
            lambda: self.api.join_chat(connection.url, session.token, token),
            lambda chat_uid: self.show_chat(server_id, chat_uid),
            on_done=lambda: self._set_chats_loading(view, False),
        )
