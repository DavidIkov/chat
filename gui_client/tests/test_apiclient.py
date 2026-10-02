"""Transport/query construction tests — no real network (task 11).

``urllib.request.urlopen`` is monkeypatched so these tests never touch a socket.
They pin the two contracts other layers rely on: the exact query string each
method builds (repeated ``uids`` keys) and the way every failure is normalised
into :class:`~chat_client.errors.APIError`.
"""

from __future__ import annotations

import unittest
import urllib.error
from unittest import mock
from urllib.parse import parse_qsl, urlparse

from chat_client.apiclient.client import ApiClient
from chat_client.apiclient.paths import BEARER_PREFIX
from chat_client.errors import APIError


class _FakeResponse:
    """Minimal stand-in for the object ``urlopen`` yields as a context manager."""

    def __init__(self, status: int = 200, body: bytes = b"") -> None:
        self._status = status
        self._body = body

    def getcode(self) -> int:
        return self._status

    def read(self) -> bytes:
        return self._body

    def __enter__(self) -> "_FakeResponse":
        return self

    def __exit__(self, *_exc: object) -> bool:
        return False


def _query_of(mock_urlopen: mock.Mock) -> list[tuple[str, str]]:
    """Return the parsed query of the single request ``urlopen`` received."""
    request = mock_urlopen.call_args[0][0]
    return parse_qsl(urlparse(request.full_url).query, keep_blank_values=True)


class QueryConstructionTests(unittest.TestCase):
    def setUp(self) -> None:
        self.client = ApiClient()

    def test_get_chats_without_uids_builds_no_query(self) -> None:
        body = b'{"chats": []}'
        for uids in (None, []):
            with self.subTest(uids=uids):
                with mock.patch(
                    "urllib.request.urlopen",
                    return_value=_FakeResponse(200, body),
                ) as urlopen:
                    self.client.get_chats("http://h", "tok", uids)
                    self.assertEqual(_query_of(urlopen), [])

    def test_get_chats_with_uids_repeats_the_key(self) -> None:
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(200, b'{"chats": []}'),
        ) as urlopen:
            self.client.get_chats("http://h", "tok", [5])
            self.assertEqual(_query_of(urlopen), [("uids", "5")])

    def test_get_chat_messages_omits_zero_limit(self) -> None:
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(200, b'{"messages": []}'),
        ) as urlopen:
            self.client.get_chat_messages("http://h", "tok", 7, limit=0)
            self.assertEqual(_query_of(urlopen), [])

    def test_get_chat_messages_includes_limit(self) -> None:
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(200, b'{"messages": []}'),
        ) as urlopen:
            self.client.get_chat_messages("http://h", "tok", 7, limit=10)
            self.assertEqual(_query_of(urlopen), [("limit", "10")])

    def test_get_users_repeats_uids(self) -> None:
        # The api_server binds []uint from repeated form values; a single
        # comma-separated value is rejected.
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(200, b'{"users": []}'),
        ) as urlopen:
            self.client.get_users("http://h", "tok", [1, 2])
            self.assertEqual(_query_of(urlopen), [("uids", "1"), ("uids", "2")])


class ErrorMappingTests(unittest.TestCase):
    def setUp(self) -> None:
        self.client = ApiClient()

    def test_error_envelope_maps_field_message_status(self) -> None:
        body = b'{"error": {"field": "name", "message": "too small user name"}}'
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(400, body),
        ):
            with self.assertRaises(APIError) as caught:
                self.client.get_chats("http://h", "tok")
        error = caught.exception
        self.assertEqual(error.field, "name")
        self.assertEqual(str(error), "too small user name")
        self.assertEqual(error.status, 400)

    def test_error_envelope_is_checked_even_on_2xx(self) -> None:
        body = b'{"error": {"field": "", "message": "nope"}}'
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(200, body),
        ):
            with self.assertRaises(APIError) as caught:
                self.client.get_chats("http://h", "tok")
        self.assertEqual(str(caught.exception), "nope")
        self.assertEqual(caught.exception.status, 200)

    def test_non_2xx_plain_text_body(self) -> None:
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(500, b"server exploded"),
        ):
            with self.assertRaises(APIError) as caught:
                self.client.get_chats("http://h", "tok")
        self.assertEqual(caught.exception.status, 500)
        self.assertEqual(str(caught.exception), "server exploded")

    def test_network_failure_maps_to_api_error_without_status(self) -> None:
        with mock.patch(
            "urllib.request.urlopen",
            side_effect=urllib.error.URLError("connection refused"),
        ):
            with self.assertRaises(APIError) as caught:
                self.client.get_chats("http://h", "tok")
        error = caught.exception
        self.assertIsNone(error.status)
        self.assertIn("Could not reach", str(error))


class AuthHeaderTests(unittest.TestCase):
    def setUp(self) -> None:
        self.client = ApiClient()

    def _authorization(self, token: str | None) -> str | None:
        with mock.patch(
            "urllib.request.urlopen",
            return_value=_FakeResponse(200, b'{"chats": []}'),
        ) as urlopen:
            self.client.get_chats("http://h", token)
            request = urlopen.call_args[0][0]
            return request.get_header("Authorization")

    def test_token_is_sent_as_bearer_header(self) -> None:
        self.assertEqual(self._authorization("tok"), BEARER_PREFIX + "tok")

    def test_no_header_without_token(self) -> None:
        self.assertIsNone(self._authorization(None))
        self.assertIsNone(self._authorization(""))


if __name__ == "__main__":  # pragma: no cover
    unittest.main()
