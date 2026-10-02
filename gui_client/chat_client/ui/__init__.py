"""Tkinter user interface: the equivalent of ``templates/`` + ``handlers/render``.

Everything here runs on the main thread. Views are dumb: they render the data the
controller hands them and forward user intent back to the controller via
``self.app.controller``. No view performs I/O.
"""

from __future__ import annotations
