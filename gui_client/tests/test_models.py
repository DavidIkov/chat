"""``from_dict`` coercion tests for the domain models (task 11)."""

from __future__ import annotations

import unittest

from chat_client.models.chat import Chat, JoinLink, Member, Message
from chat_client.models.shared import timestamp_from_json, uid_from_json
from chat_client.models.user import User, UserSession


class JsonCoercionTests(unittest.TestCase):
    """The shared scalar helpers treat missing/empty values as ``0``."""

    def test_uid_from_json(self) -> None:
        self.assertEqual(uid_from_json(None), 0)
        self.assertEqual(uid_from_json(""), 0)
        self.assertEqual(uid_from_json("8"), 8)
        self.assertEqual(uid_from_json(8), 8)

    def test_timestamp_from_json(self) -> None:
        self.assertEqual(timestamp_from_json(None), 0)
        self.assertEqual(timestamp_from_json(""), 0)
        self.assertEqual(timestamp_from_json("123"), 123)


class UserModelTests(unittest.TestCase):
    def test_from_dict_tolerates_missing_keys(self) -> None:
        user = User.from_dict({})
        self.assertEqual((user.uid, user.name), (0, ""))

    def test_from_dict_coerces_types(self) -> None:
        user = User.from_dict({"uid": "3", "name": 5})
        self.assertEqual(user.uid, 3)
        self.assertEqual(user.name, "5")

    def test_session_round_trips(self) -> None:
        session = UserSession.from_dict({"uid": "4", "token": "abc"})
        self.assertEqual(session, UserSession(uid=4, token="abc"))
        self.assertEqual(
            UserSession.from_dict(session.to_dict()), session
        )

    def test_session_missing_token(self) -> None:
        session = UserSession.from_dict({})
        self.assertEqual((session.uid, session.token), (0, ""))


class ChatModelTests(unittest.TestCase):
    def test_chat_defaults(self) -> None:
        chat = Chat.from_dict({})
        self.assertEqual(chat.chat_uid, 0)
        self.assertEqual(chat.name, "")
        self.assertEqual(chat.created_at, 0)
        self.assertEqual(chat.creator_user_uid, 0)

    def test_chat_coerces(self) -> None:
        chat = Chat.from_dict(
            {
                "chat_uid": "7",
                "name": "general",
                "created_at": None,
                "creator_user_uid": "",
            }
        )
        self.assertEqual(chat.chat_uid, 7)
        self.assertEqual(chat.name, "general")
        self.assertEqual(chat.created_at, 0)
        self.assertEqual(chat.creator_user_uid, 0)

    def test_message_defaults(self) -> None:
        message = Message.from_dict({})
        self.assertEqual(message.message_uid, 0)
        self.assertEqual(message.text, "")
        self.assertEqual(message.user_uid, 0)
        self.assertEqual(message.chat_uid, 0)
        self.assertEqual(message.created_at, 0)

    def test_message_coerces(self) -> None:
        message = Message.from_dict(
            {"message_uid": "9", "text": "hi", "user_uid": "2", "chat_uid": None}
        )
        self.assertEqual(message.message_uid, 9)
        self.assertEqual(message.text, "hi")
        self.assertEqual(message.user_uid, 2)
        self.assertEqual(message.chat_uid, 0)

    def test_member_defaults_and_coercion(self) -> None:
        self.assertEqual(Member.from_dict({}).user_uid, 0)
        self.assertEqual(Member.from_dict({"user_uid": "5"}).user_uid, 5)

    def test_join_link(self) -> None:
        self.assertEqual(JoinLink.from_dict({}), JoinLink(token="", expires_at=0))
        link = JoinLink.from_dict({"token": "abc", "expires_at": "123"})
        self.assertEqual(link.token, "abc")
        self.assertEqual(link.expires_at, 123)


if __name__ == "__main__":  # pragma: no cover
    unittest.main()
