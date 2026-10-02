"""Chat-page operations — parity with the chat/message parts of
``internal/webui_server/handlers/chats``.

Covers FR-9..12: reading a chat, sending messages, creating join links and
leaving. Mixed into :class:`~chat_client.controller.controller.Controller`.

``load_chat`` is the most involved operation: it makes **four** api_server calls
(exactly like ``renderChatPage`` in Go) and assembles the view data. It mirrors
one WebUI handler each:

* :meth:`load_chat` ← ``ChatPageHandler`` / ``renderChatPage``;
* :meth:`send_message` ← ``SendMessageHandler``;
* :meth:`create_join_link` ← ``CreateJoinLinkHandler``;
* :meth:`leave_chat` ← ``LeaveChatHandler``.

Pending join link: ``create_join_link`` stores the token in
``state.set_pending_join_link`` and then calls ``load_chat``; ``load_chat``
consumes it with ``state.take_pending_join_link`` and passes it to the view. This
reproduces the WebUI's show-once behaviour (FR-11) without HTTP redirects.
"""

from __future__ import annotations

from typing import Any

from chat_client.constants import MESSAGE_TEXT_MAX, MESSAGE_TEXT_MIN
from chat_client.errors import APIError, AppError, ValidationError
from chat_client.models.chat import Chat, Member, Message
from chat_client.ui.views.chat import ChatMemberItem, ChatMessageItem


class ChatNotFound(AppError):
    """``GET /chat/get_chats?uids=…`` returned nothing for the requested chat.

    The api_server filters by membership, so an empty result means the chat does
    not exist *or* the caller is not a member. Both are a 404 in the WebUI and map
    to the error view here.
    """

    def __init__(self) -> None:
        super().__init__("chat not found")


class ChatMixin:
    """One chat's page operations."""

    # --- helpers -----------------------------------------------------------

    def _chat_view(self) -> Any:
        """Return the currently mounted view, or ``None`` (see ``ChatsMixin``)."""
        return getattr(self.app, "_view", None)

    def _set_chat_loading(self, view: Any, loading: bool) -> None:
        """Toggle ``view``'s busy indicator, but only while it is still mounted."""
        if view is None or self._chat_view() is not view:
            return
        setter = getattr(view, "set_loading", None)
        if setter is not None:
            setter(loading)

    def _load_error(self, error: BaseException) -> None:
        """Send any chat-loading failure to the error view (the WebUI's 502/404)."""
        self.show_error(str(error) or error.__class__.__name__)

    # --- operations --------------------------------------------------------

    def load_chat(self, server_id: int, chat_uid: int) -> None:
        """Load and render one chat (FR-9).

        Steps, in order (matching ``renderChatPage``):

        1. resolve a signed-in connection (else navigate away);
        2. ``GET /chat/get_chats?uids={chat_uid}`` for the name; empty result ⇒
           error view "chat not found" (the backend filters by membership);
        3. ``GET /chat/{uid}/get_messages`` (no limit);
        4. ``GET /chat/{uid}/get_members``;
        5. one batch ``GET /user/get?uids=…`` for all member uids (names are
           best-effort: a failure leaves names blank, as in Go);
        6. consume any pending join link for this chat;
        7. hand ``(chat, messages, members, join_link)`` to the mounted view.

        The four calls run in one worker so the sequence stays readable; every
        failure (including a missing chat) goes to the error view, since a page
        load cannot be corrected by editing a field.
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        view = self._chat_view()
        self._set_chat_loading(view, True)
        url, token = connection.url, session.token

        def work() -> tuple[Chat, list[Message], list[Member], dict[int, str]]:
            chats = self.api.get_chats(url, token, [chat_uid])
            if not chats:
                raise ChatNotFound()
            messages = self.api.get_chat_messages(url, token, chat_uid)
            members = self.api.get_chat_members(url, token, chat_uid)
            names: dict[int, str] = {}
            uids = [member.user_uid for member in members]
            if uids:
                try:
                    names = {
                        user.uid: user.name
                        for user in self.api.get_users(url, token, uids)
                    }
                except APIError:
                    # Names are cosmetic; a resolver failure must not fail the page.
                    names = {}
            return chats[0], messages, members, names

        def on_loaded(
            bundle: tuple[Chat, list[Message], list[Member], dict[int, str]]
        ) -> None:
            chat, messages, members, names = bundle
            # Consume the one-shot token even if the view has since gone: the
            # WebUI consumes it on the GET that follows the join-link POST.
            join_link = self.state.take_pending_join_link(server_id, chat_uid)
            if view is None or self._chat_view() is not view:
                return
            render = getattr(view, "render_chat", None)
            if render is None:
                return
            render(
                chat,
                [
                    ChatMessageItem(message, names.get(message.user_uid, ""))
                    for message in messages
                ],
                [
                    ChatMemberItem(member.user_uid, names.get(member.user_uid, ""))
                    for member in members
                ],
                join_link,
            )

        self._run(
            work,
            on_loaded,
            on_error=self._load_error,
            on_done=lambda: self._set_chat_loading(view, False),
        )

    def send_message(self, server_id: int, chat_uid: int, text: str) -> None:
        """``POST /chat/{uid}/send_message`` then reload the chat (FR-10).

        The text is validated locally first (1–256 characters, trimmed like the
        Go validator). On success the chat is reloaded (the PRG equivalent); on
        failure the message is shown in the page banner and the user stays.
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        url, token = connection.url, session.token
        text = text.strip()
        try:
            if len(text) < MESSAGE_TEXT_MIN:
                raise ValidationError("a message cannot be empty", field="text")
            if len(text) > MESSAGE_TEXT_MAX:
                raise ValidationError(
                    f"a message can have at most {MESSAGE_TEXT_MAX} characters",
                    field="text",
                )
        except ValidationError as exc:
            self.app.show_banner_error(str(exc))
            return
        view = self._chat_view()
        self._set_chat_loading(view, True)

        def on_sent(_message_uid: Any) -> None:
            # Stop this request's busy state *before* the reload starts its own,
            # so the reload's indicator is not cancelled by our on_done.
            self._set_chat_loading(view, False)
            self.load_chat(server_id, chat_uid)

        def on_error(error: BaseException) -> None:
            self._set_chat_loading(view, False)
            self._show_api_error(error)

        self._run(
            lambda: self.api.send_message(url, token, chat_uid, text),
            on_sent,
            on_error=on_error,
        )

    def create_join_link(
        self,
        server_id: int,
        chat_uid: int,
        lifetime_seconds: int,
        max_uses: int,
    ) -> None:
        """``POST /chat/{uid}/create_join_link`` then reload to show the token (FR-11).

        ``0`` for either limit means "no limit"/"never expires". The caller (the
        view) is responsible for only passing a limit when its checkbox is ticked
        and for validating it locally. The token is stashed on the state and the
        chat reloaded; ``load_chat`` consumes it so it is shown exactly once.
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        url, token = connection.url, session.token
        view = self._chat_view()
        self._set_chat_loading(view, True)

        def on_created(link: Any) -> None:
            self._set_chat_loading(view, False)
            self.state.set_pending_join_link(server_id, chat_uid, link.token)
            self.load_chat(server_id, chat_uid)

        def on_error(error: BaseException) -> None:
            self._set_chat_loading(view, False)
            self._show_api_error(error)

        self._run(
            lambda: self.api.create_join_link(
                url,
                token,
                chat_uid,
                lifetime_seconds=lifetime_seconds,
                max_uses=max_uses,
            ),
            on_created,
            on_error=on_error,
        )

    def leave_chat(self, server_id: int, chat_uid: int) -> None:
        """``POST /chat/{uid}/leave`` then return to the chat list (FR-12).

        A backend failure is a 502 in the WebUI, so it goes to the error view.
        """
        connection = self._signed_in_server(server_id)
        if connection is None:
            return
        session = connection.session
        if session is None:  # pragma: no cover - _signed_in_server checked this
            return
        view = self._chat_view()
        self._set_chat_loading(view, True)

        def on_left(_result: Any) -> None:
            self._set_chat_loading(view, False)
            self.show_chats(server_id)

        def on_error(error: BaseException) -> None:
            self._set_chat_loading(view, False)
            self.show_error(str(error) or error.__class__.__name__)

        self._run(
            lambda: self.api.leave_chat(connection.url, session.token, chat_uid),
            on_left,
            on_error=on_error,
        )
