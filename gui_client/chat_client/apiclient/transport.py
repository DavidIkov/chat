"""Low-level JSON-over-HTTP transport shared by every api_server method.

This is the Python equivalent of ``apiclient.do`` in
``internal/webui_server/services/apiclient/client.go``. It exists so the
per-domain mixins (:mod:`chat_client.apiclient.users`,
:mod:`chat_client.apiclient.chats`) contain only endpoint knowledge and never
touch ``urllib`` directly.

The implementation is standard-library only (``urllib.request``), matching the
project's dependency-free posture.
"""

from __future__ import annotations

import json
from collections.abc import Mapping, Sequence
from http import HTTPStatus
from typing import Any
from urllib import error as urllib_error
from urllib import parse as urllib_parse
from urllib import request as urllib_request

from chat_client.apiclient.paths import BEARER_PREFIX
from chat_client.constants import HTTP_TIMEOUT_SECONDS
from chat_client.errors import APIError
from chat_client.models.shared import uid_from_json


def as_list(response: Any, key: str) -> list[Mapping[str, Any]]:
    """Return ``response[key]`` as a list of mapping rows.

    A missing key (``omitempty``) or a non-list value yields ``[]``; non-mapping
    elements are dropped so ``Model.from_dict`` only ever sees dictionaries.
    """
    if not isinstance(response, Mapping):
        return []
    value = response.get(key)
    if not isinstance(value, list):
        return []
    return [item for item in value if isinstance(item, Mapping)]


def as_uid(response: Any, key: str) -> int:
    """Return ``response[key]`` coerced to a uid (``0`` when absent/empty)."""
    if not isinstance(response, Mapping):
        return 0
    return uid_from_json(response.get(key))


def _reason_phrase(status: int) -> str:
    """Best-effort HTTP reason phrase for a status code."""
    try:
        return HTTPStatus(status).phrase
    except ValueError:
        return f"HTTP {status}"


class Transport:
    """Sends one JSON request at a time and normalises every failure.

    :param timeout: per-request timeout in seconds; ``None`` disables it.
    """

    def __init__(self, timeout: float | None = HTTP_TIMEOUT_SECONDS) -> None:
        self.timeout = timeout

    def request(
        self,
        method: str,
        base_url: str,
        path: str,
        *,
        query: Mapping[str, Sequence[str] | str] | None = None,
        token: str | None = None,
        body: Any | None = None,
    ) -> Any:
        """Send a request and return the decoded JSON response.

        Contract (mirrors ``do``):

        * ``target = base_url.rstrip('/') + path``, plus ``urlencode(query)``
          when ``query`` is non-empty;
        * when ``body`` is not ``None``, JSON-encode it and send
          ``Content-Type: application/json``;
        * when ``token`` is non-empty, send ``Authorization: Bearer <token>``;
        * decode the body as JSON; an ``{"error": {"field", "message"}}`` payload
          or a non-2xx status raises :class:`~chat_client.errors.APIError` with
          ``status``/``field``/``message`` filled in;
        * return ``None`` for an empty body, otherwise the parsed JSON value.

        Network failures (DNS, refused connection, timeout, TLS) are wrapped in
        :class:`~chat_client.errors.APIError` with ``status=None`` so callers
        only ever handle one exception type.
        """
        target = base_url.rstrip("/") + path
        if query:
            encoded = urllib_parse.urlencode(query, doseq=True)
            if encoded:
                target += "?" + encoded

        data: bytes | None = None
        headers: dict[str, str] = {}
        if body is not None:
            data = json.dumps(body).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if token:
            headers["Authorization"] = BEARER_PREFIX + token

        request = urllib_request.Request(target, data=data, headers=headers, method=method)

        status: int
        raw: bytes
        try:
            with urllib_request.urlopen(request, timeout=self.timeout) as response:
                status = response.getcode()
                raw = response.read()
        except urllib_error.HTTPError as exc:
            # A real response with a non-2xx status: keep the body so the
            # envelope/status handling below can describe the failure.
            status = exc.code
            raw = exc.read()
        except OSError as exc:
            # URLError, socket timeouts and TLS errors are all OSError subclasses.
            reason = getattr(exc, "reason", exc)
            raise APIError(f"Could not reach {base_url}: {reason}", status=None) from exc

        payload = self._decode(raw)

        # Best effort: the api_server returns {"error":{...}} for structured
        # failures, sometimes with a 2xx status. This is checked first, exactly
        # like ``do``.
        if isinstance(payload, Mapping):
            error = payload.get("error")
            if isinstance(error, Mapping):
                message = str(error.get("message", "")) or _reason_phrase(status)
                field = str(error.get("field", "") or "")
                raise APIError(message, status=status, field=field)

        if not 200 <= status < 300:
            text = raw.decode("utf-8", "replace").strip()
            raise APIError(text or _reason_phrase(status), status=status)

        return payload

    @staticmethod
    def _decode(raw: bytes) -> Any:
        """Decode a JSON body, falling back to ``None`` for an empty/invalid one."""
        if not raw:
            return None
        try:
            return json.loads(raw)
        except ValueError:
            return None

    def close(self) -> None:
        """Release any held resources. Safe to call more than once.

        Requests use one-shot ``urlopen`` calls, so there is nothing to release
        today; the hook exists for symmetry with :class:`ApiClient` and future
        connection pooling.
        """
        return None
