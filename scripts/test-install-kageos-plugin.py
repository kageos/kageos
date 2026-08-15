#!/usr/bin/env python3
"""在隔离目录中验证官网安装包和安装脚本。"""

from __future__ import annotations

import json
import os
import subprocess
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
WEBSITE_DOWNLOADS = ROOT.parent / "kageos-website" / "public" / "downloads" / "kageos"


def main() -> int:
    with tempfile.TemporaryDirectory() as directory:
        isolated = Path(directory)
        environment = os.environ.copy()
        environment.update(
            {
                "KAGEOS_PLUGIN_BASE_URL": WEBSITE_DOWNLOADS.resolve().as_uri(),
                "KAGEOS_PLUGIN_HOME": str(isolated / "plugins"),
                "KAGEOS_MARKETPLACE_FILE": str(isolated / "marketplace.json"),
            }
        )
        subprocess.run(["bash", str(ROOT / "scripts" / "install-kageos-plugin.sh")], env=environment, check=True)
        manifest = json.loads(
            (isolated / "plugins" / "kageos" / ".codex-plugin" / "plugin.json").read_text(encoding="utf-8")
        )
        marketplace = json.loads((isolated / "marketplace.json").read_text(encoding="utf-8"))
        latest = json.loads((WEBSITE_DOWNLOADS / "latest.json").read_text(encoding="utf-8"))
        if manifest.get("version") != latest.get("version"):
            raise ValueError("安装版本与 latest.json 不一致")
        if not any(item.get("name") == "kageos" for item in marketplace.get("plugins", [])):
            raise ValueError("Personal marketplace 中没有 kageos")
    print("安装器冒烟测试通过")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
