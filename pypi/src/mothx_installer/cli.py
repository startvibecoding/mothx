import os
import subprocess
import sys
from pathlib import Path


def _binary_name() -> str:
    if sys.platform.startswith("win"):
        return "mothx.exe"
    return "mothx"


def _binary_path() -> Path:
    package_dir = Path(__file__).resolve().parent
    preferred = package_dir / "bin" / _binary_name()
    if preferred.is_file():
        return preferred
    legacy = "vibecoding.exe" if sys.platform.startswith("win") else "vibecoding"
    return package_dir / "bin" / legacy


def main() -> int:
    binary = _binary_path()
    if not binary.is_file():
        sys.stderr.write("MothX binary is missing from this PyPI installation.\n")
        sys.stderr.write("Reinstall with: pip install --force-reinstall mothx-installer\n")
        return 1

    args = [str(binary), *sys.argv[1:]]
    if not sys.platform.startswith("win"):
        try:
            os.execv(str(binary), args)
        except OSError as exc:
            sys.stderr.write(f"Failed to execute MothX binary: {exc}\n")
            return 1
        return 1

    return subprocess.call(args)
