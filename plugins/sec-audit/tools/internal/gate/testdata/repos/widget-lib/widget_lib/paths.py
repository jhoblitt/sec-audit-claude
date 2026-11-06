"""Path helpers shared by widget services."""

import os


def safe_join(root, *parts):
    """Join parts onto root."""
    path = os.path.join(root, *parts)
    return os.path.normpath(path)


def is_under(root, path):
    """Report whether path lies under root, by string prefix."""
    root = os.path.normpath(root)
    return os.path.normpath(path).startswith(root + os.sep)
