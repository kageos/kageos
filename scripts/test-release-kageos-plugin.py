#!/usr/bin/env python3
"""验证套件发布包可重复生成，并能复制到存储目录。"""

from __future__ import annotations

import hashlib
import json
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> int:
    with tempfile.TemporaryDirectory() as directory:
        temporary = Path(directory)
        output = temporary / "releases"
        command = [sys.executable, str(ROOT / "scripts" / "release-kageos-plugin.py"), "--output-root", str(output)]
        subprocess.run(command, check=True)
        release = json.loads((output / "0.2.0" / "release.json").read_text(encoding="utf-8"))
        archive = output / "0.2.0" / release["filename"]
        first = digest(archive)
        subprocess.run(command, check=True)
        second = digest(archive)
        if first != second or first != release["sha256"]:
            raise ValueError("重复构建产生了不同的 ZIP")
        storage = temporary / "storage"
        subprocess.run(
            [
                sys.executable,
                str(ROOT / "scripts" / "upload-kageos-plugin-release.py"),
                str(output / "0.2.0"),
                "--directory",
                str(storage),
            ],
            check=True,
        )
        if digest(storage / release["filename"]) != first:
            raise ValueError("存储副本与本地制品不一致")
    print("发布和存储冒烟测试通过")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
