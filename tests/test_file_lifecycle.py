import os
import tempfile
import time
import pytest
from file_lifecycle import FileLifecycleManager


@pytest.fixture
def manager():
    return FileLifecycleManager()


class TestRegisterFile:
    def test_pending_contains_registered_path(self, manager, tmp_path):
        f = tmp_path / "test.txt"
        f.write_text("x")
        manager.register_file(str(f), delay=9999)
        assert str(f) in manager.pending()

    def test_file_deleted_after_delay(self, manager, tmp_path):
        f = tmp_path / "test.txt"
        f.write_text("x")
        manager.register_file(str(f), delay=0)
        time.sleep(0.2)
        assert not f.exists()

    def test_pending_cleared_after_deletion(self, manager, tmp_path):
        f = tmp_path / "test.txt"
        f.write_text("x")
        manager.register_file(str(f), delay=0)
        time.sleep(0.2)
        assert str(f) not in manager.pending()


class TestRegisterDir:
    def test_dir_deleted_after_delay(self, manager, tmp_path):
        d = tmp_path / "subdir"
        d.mkdir()
        (d / "file.txt").write_text("x")
        manager.register_dir(str(d), delay=0)
        time.sleep(0.2)
        assert not d.exists()


class TestPending:
    def test_pending_returns_list(self, manager):
        assert isinstance(manager.pending(), list)

    def test_multiple_registrations(self, manager, tmp_path):
        paths = []
        for i in range(3):
            f = tmp_path / f"test{i}.txt"
            f.write_text("x")
            manager.register_file(str(f), delay=9999)
            paths.append(str(f))
        pending = manager.pending()
        for p in paths:
            assert p in pending
