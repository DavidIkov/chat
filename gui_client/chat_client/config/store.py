"""Local configuration persistence.

The desktop client keeps its server connections (and the bearer tokens for each)
on disk so they survive a restart. This is the one deliberate behavioural
difference from the Go WebUI, which is in-memory only (see the client
requirements, decision C3).

On-disk schema (JSON, ``version = 1``)::

    {
      "version": 1,
      "connections": [
        {
          "id": 1,
          "url": "http://localhost:8080",
          "name": "Local",
          "session": {"uid": 1, "token": "..."}
        }
      ],
      "next_id": 1
    }

The file is written atomically (temp file + ``os.replace``) with mode ``0600``
because it contains bearer tokens.
"""

from __future__ import annotations

import json
import os
import tempfile
from pathlib import Path

from chat_client.constants import (
    CONFIG_DIR_MODE,
    CONFIG_DIR_NAME,
    CONFIG_FILE_MODE,
    CONFIG_FILE_NAME,
    CONFIG_VERSION,
)
from chat_client.errors import ConfigError


def default_config_path() -> Path:
    """Return the config file path.

    Honours ``XDG_CONFIG_HOME`` when set, otherwise ``~/.config`` (Linux/macOS
    convention). Windows support can be added later behind this function so
    nothing else has to change.
    """
    base = os.environ.get("XDG_CONFIG_HOME")
    if not base:
        base = str(Path.home() / ".config")
    return Path(base) / CONFIG_DIR_NAME / CONFIG_FILE_NAME


class ConfigStore:
    """Loads and saves the JSON config file for the app.

    :param path: explicit config path (used by ``--config`` and tests); when
        ``None``, :func:`default_config_path` is used.
    """

    def __init__(self, path: Path | None = None) -> None:
        self.path = Path(path) if path is not None else default_config_path()

    def load(self) -> dict:
        """Return the parsed config, or a fresh empty config when absent.

        A missing file is not an error (return the empty schema). A file that
        exists but is unreadable or malformed raises
        :class:`~chat_client.errors.ConfigError` — the caller decides whether to
        back it up and start fresh.
        """
        if not self.path.exists():
            return {"version": CONFIG_VERSION, "next_id": 0, "connections": []}

        try:
            text = self.path.read_text(encoding="utf-8")
        except (OSError, ValueError) as exc:
            raise ConfigError(f"could not read config file {self.path}: {exc}") from exc

        try:
            data = json.loads(text)
        except ValueError as exc:
            raise ConfigError(
                f"config file {self.path} is not valid JSON: {exc}"
            ) from exc

        if not isinstance(data, dict):
            raise ConfigError(f"config file {self.path} must contain a JSON object")
        return data

    def save(self, data: dict) -> None:
        """Atomically persist ``data``, creating the directory if needed.

        The directory is created with mode ``0700`` and the file with ``0600``.
        An existing file keeps its restrictive mode via ``os.replace`` on a
        freshly-created temp file.
        """
        payload = dict(data)
        payload["version"] = CONFIG_VERSION

        directory = self.path.parent
        tmp_path: Path | None = None
        try:
            directory.mkdir(parents=True, exist_ok=True, mode=CONFIG_DIR_MODE)
            os.chmod(directory, CONFIG_DIR_MODE)

            # The temp file must live in the same directory so os.replace is an
            # atomic rename within one filesystem.
            fd, tmp_name = tempfile.mkstemp(
                dir=str(directory), prefix=self.path.name + ".", suffix=".tmp"
            )
            tmp_path = Path(tmp_name)
            with os.fdopen(fd, "w", encoding="utf-8") as handle:
                json.dump(payload, handle, indent=2, ensure_ascii=False)
                handle.write("\n")
                handle.flush()
                os.fsync(handle.fileno())

            # mkstemp already uses 0600; set it explicitly to defeat any umask.
            os.chmod(tmp_path, CONFIG_FILE_MODE)
            os.replace(tmp_path, self.path)
        except (OSError, TypeError, ValueError) as exc:
            raise ConfigError(f"could not write config file {self.path}: {exc}") from exc
        finally:
            if tmp_path is not None and tmp_path.exists():
                try:
                    tmp_path.unlink()
                except OSError:
                    pass

    def clear(self) -> None:
        """Delete the config file if it exists (used by "forget everything")."""
        try:
            self.path.unlink()
        except FileNotFoundError:
            return
        except OSError as exc:
            raise ConfigError(
                f"could not remove config file {self.path}: {exc}"
            ) from exc
