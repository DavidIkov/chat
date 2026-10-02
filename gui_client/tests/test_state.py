"""State transitions, URL validation and persistence tests (task 11)."""

from __future__ import annotations

import unittest

from chat_client.errors import InvalidServerUrlError, ServerNotFoundError
from chat_client.models.user import UserSession
from chat_client.state.persistence import state_from_config, state_to_config
from chat_client.state.session import AppState, validate_server_url


class ValidateServerUrlTests(unittest.TestCase):
    def test_accepts_http_and_https(self) -> None:
        self.assertEqual(
            validate_server_url("http://localhost:8080"), "http://localhost:8080"
        )
        self.assertEqual(validate_server_url("https://example.com"), "https://example.com")

    def test_strips_trailing_slash(self) -> None:
        self.assertEqual(
            validate_server_url("http://localhost:8080///"), "http://localhost:8080"
        )

    def test_rejects_bad_scheme_and_host(self) -> None:
        for bad in ("ftp://example.com", "http://", "", "not a url"):
            with self.subTest(url=bad):
                with self.assertRaises(InvalidServerUrlError):
                    validate_server_url(bad)


class AppStateServerTests(unittest.TestCase):
    def test_add_server_assigns_increasing_ids(self) -> None:
        state = AppState()
        first = state.add_server("http://one", "One")
        second = state.add_server("http://two")
        self.assertEqual((first.id, second.id), (1, 2))
        self.assertEqual(state.next_id, 2)
        self.assertEqual(len(state.list_servers()), 2)

    def test_server_by_id_raises_when_missing(self) -> None:
        state = AppState()
        with self.assertRaises(ServerNotFoundError):
            state.server_by_id(42)

    def test_remove_server(self) -> None:
        state = AppState()
        connection = state.add_server("http://one")
        state.remove_server(connection.id)
        self.assertEqual(state.list_servers(), [])
        self.assertIsNone(state.find_server(connection.id))

    def test_label_falls_back_to_url(self) -> None:
        state = AppState()
        named = state.add_server("http://one", "One")
        unnamed = state.add_server("http://two")
        self.assertEqual(named.label, "One")
        self.assertEqual(unnamed.label, "http://two")


class AppStateSessionTests(unittest.TestCase):
    def test_set_and_clear_session(self) -> None:
        state = AppState()
        connection = state.add_server("http://one")
        self.assertFalse(connection.signed_in)

        state.set_server_session(connection.id, UserSession(uid=1, token="t"))
        self.assertTrue(connection.signed_in)
        self.assertEqual(connection.session, UserSession(uid=1, token="t"))

        state.clear_server_session(connection.id)
        self.assertFalse(connection.signed_in)


class PendingJoinLinkTests(unittest.TestCase):
    def test_take_returns_once_then_empty(self) -> None:
        state = AppState()
        state.set_pending_join_link(1, 7, "token-a")
        self.assertEqual(state.take_pending_join_link(1, 7), "token-a")
        self.assertEqual(state.take_pending_join_link(1, 7), "")

    def test_mismatch_leaves_pending_intact(self) -> None:
        state = AppState()
        state.set_pending_join_link(1, 7, "token-a")
        self.assertEqual(state.take_pending_join_link(2, 7), "")
        self.assertEqual(state.take_pending_join_link(1, 8), "")
        self.assertEqual(state.take_pending_join_link(1, 7), "token-a")


class PersistenceTests(unittest.TestCase):
    def _sample_state(self) -> AppState:
        state = AppState()
        first = state.add_server("http://one", "One")
        state.set_server_session(first.id, UserSession(uid=3, token="tok"))
        state.add_server("http://two")
        return state

    def test_round_trip_preserves_everything(self) -> None:
        state = self._sample_state()
        restored = state_from_config(state_to_config(state))

        self.assertEqual(restored.next_id, state.next_id)
        self.assertEqual(
            [(c.id, c.url, c.name) for c in restored.connections],
            [(c.id, c.url, c.name) for c in state.connections],
        )
        self.assertEqual(restored.connections[0].session, UserSession(uid=3, token="tok"))
        self.assertIsNone(restored.connections[1].session)

    def test_empty_document(self) -> None:
        restored = state_from_config({})
        self.assertEqual(restored.connections, [])
        self.assertEqual(restored.next_id, 0)

    def test_malformed_entries_do_not_raise(self) -> None:
        restored = state_from_config(
            {
                "connections": [None, 5, {"url": "http://x"}, {"id": 2}],
                "next_id": "nonsense",
            }
        )
        self.assertEqual(restored.connections, [])
        self.assertEqual(restored.next_id, 0)

    def test_invalid_next_id_never_below_highest_connection(self) -> None:
        restored = state_from_config(
            {
                "connections": [{"id": 9, "url": "http://x", "name": "", "session": None}],
                "next_id": 2,
            }
        )
        self.assertEqual(restored.connections[0].id, 9)
        self.assertEqual(restored.next_id, 9)


if __name__ == "__main__":  # pragma: no cover
    unittest.main()
