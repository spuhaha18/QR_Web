"""
File lifecycle management: register files and directories for deferred cleanup.
"""
import os
import shutil
import threading
import time
import logging

logger = logging.getLogger(__name__)


class FileLifecycleManager:
    """Manages deferred cleanup of temporary files and directories.

    Registered paths are deleted after a configurable delay in background threads.
    A registry of pending cleanups is maintained for observability.
    """

    def __init__(self):
        self._pending: dict[str, threading.Thread] = {}
        self._lock = threading.Lock()

    def register_file(self, path: str, delay: int = 600) -> None:
        """Schedule a file for deletion after `delay` seconds."""
        self._schedule(path, delay, is_dir=False)

    def register_dir(self, path: str, delay: int = 60) -> None:
        """Schedule a directory for deletion after `delay` seconds."""
        self._schedule(path, delay, is_dir=True)

    def pending(self) -> list[str]:
        """Return list of paths currently scheduled for deletion."""
        with self._lock:
            return list(self._pending.keys())

    def _schedule(self, path: str, delay: int, is_dir: bool) -> None:
        def cleanup():
            time.sleep(delay)
            try:
                if is_dir:
                    shutil.rmtree(path, ignore_errors=True)
                else:
                    if os.path.exists(path):
                        os.remove(path)
            except OSError as e:
                logger.warning("Failed to clean up %s: %s", path, e)
            finally:
                with self._lock:
                    self._pending.pop(path, None)

        t = threading.Thread(target=cleanup, daemon=True)
        with self._lock:
            self._pending[path] = t
        t.start()


file_lifecycle = FileLifecycleManager()
