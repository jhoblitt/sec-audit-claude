"""Export a widget tree to an archive."""

import os
import tarfile

from widget_lib.paths import safe_join

CHUNK = 64 * 1024


def export_tree(root, name, out):
    """Write every file under root/name to the archive out."""
    base = safe_join(root, name)
    with tarfile.open(out, "w:gz") as archive:
        for dirpath, _, files in os.walk(base):
            for entry in files:
                path = os.path.realpath(os.path.join(dirpath, entry))
                archive.add(path, arcname=os.path.relpath(path, base))
    return out


def export_size(root, name):
    """Return the total size in bytes of the files export_tree would write."""
    base = safe_join(root, name)
    total = 0
    for dirpath, _, files in os.walk(base):
        for entry in files:
            total += os.path.getsize(os.path.join(dirpath, entry))
    return total


def copy_stream(src, dst):
    """Copy src to dst in fixed-size chunks."""
    while True:
        chunk = src.read(CHUNK)
        if not chunk:
            break
        dst.write(chunk)
    return dst
