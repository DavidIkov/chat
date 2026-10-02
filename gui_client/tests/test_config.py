"""Atomic config store tests: schema, mode and corruption (task 11)."""

from __future__ import annotations

import json
import stat
import tempfile
import unittest
from pathlib import Path

from chat_client.config.store import ConfigStore
from chat_client.constants import CONFIG_DIR_MODE, CONFIG_FILE_MODE, CONFIG_VERSION
from chat_client.errors import ConfigError


class ConfigStoreTests(unittest.TestCase):
    def setUp(self) -> None:
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = Path(self._tmp.name)
        # A nested path proves the store creates its parent directory.
        self.path = self.root / "nested" / "gui_client.json"
        self.store = ConfigStore(self.path)

    def test_missing_file_returns_empty_schema(self) -> None:
        data = self.store.load()
        self.assertEqual(
            data, {"version": CONFIG_VERSION, "next_id": 0, "connections": []}
        )

    def test_save_load_round_trip(self) -> None:
        payload = {
            "next_id": 1,
            "connections": [
                {"id": 1, "url": "http://one", "name": "One", "session": None}
            ],
        }
        self.store.save(payload)

        loaded = self.store.load()
        self.assertEqual(loaded["version"], CONFIG_VERSION)
        self.assertEqual(loaded["next_id"], 1)
        self.assertEqual(loaded["connections"], payload["connections"])

    def test_file_mode_is_0600_and_dir_created_0700(self) -> None:
        self.store.save({"next_id": 0, "connections": []})
        file_mode = stat.S_IMODE(self.path.stat().st_mode)
        dir_mode = stat.S_IMODE(self.path.parent.stat().st_mode)
        self.assertEqual(file_mode, CONFIG_FILE_MODE)
        self.assertEqual(dir_mode, CONFIG_DIR_MODE)

    def test_corrupt_json_raises_config_error(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.path.write_text("{ this is not json", encoding="utf-8")
        with self.assertRaises(ConfigError):
            self.store.load()

    def test_non_object_json_raises_config_error(self) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self.path.write_text(json.dumps([1, 2, 3]), encoding="utf-8")
        with self.assertRaises(ConfigError):
            self.store.load()

    def test_no_tmp_leftovers(self) -> None:
        self.store.save({"next_id": 0, "connections": []})
        self.store.save({"next_id": 1, "connections": []})
        leftovers = list(self.path.parent.glob("*.tmp"))
        self.assertEqual(leftovers, [])

    def test_clear_removes_file(self) -> None:
        self.store.save({"next_id": 0, "connections": []})
        self.assertTrue(self.path.exists())
        self.store.clear()
        self.assertFalse(self.path.exists())
        # Clearing a missing file is a no-op, not an error.
        self.store.clear()


if __name__ == "__main__":  # pragma: no cover
    unittest.main()
