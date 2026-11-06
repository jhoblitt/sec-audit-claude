"""An in-process cache of rendered widgets."""

import hashlib

_entries = {}


def key(name, params):
    """Return the cache key for a widget rendered with params."""
    digest = hashlib.sha256(name.encode()).hexdigest()
    return digest[:8]


def get(name, params):
    """Return the cached rendering, or None."""
    return _entries.get(key(name, params))


def put(name, params, rendering):
    """Store a rendering under its key."""
    _entries[key(name, params)] = rendering


def clear():
    """Drop every cached rendering."""
    _entries.clear()
