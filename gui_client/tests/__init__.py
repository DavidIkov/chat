"""Unit tests for the pure (no-Tk) layers.

Run headlessly with::

    cd gui_client && python3 -m unittest discover -s tests -v

These tests are deliberately not packaged (see ``pyproject.toml``); they only
exercise ``models``, ``state``, ``config`` and ``apiclient``, none of which touch
Tkinter or the real network.
"""

from __future__ import annotations
