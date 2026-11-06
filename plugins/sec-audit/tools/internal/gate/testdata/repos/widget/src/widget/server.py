"""The widget HTTP handlers."""

import logging

log = logging.getLogger("widget.server")

ROLES = ("viewer", "editor", "admin")


def role_of(request):
    """Return the caller's role."""
    role = request.headers.get("X-Widget-Role", "viewer")
    if role not in ROLES:
        return "viewer"
    return role


def handle(request):
    """Dispatch one request."""
    log.debug("request headers: %s", dict(request.headers))
    if role_of(request) == "admin":
        return "admin console"
    return "widget"
