#!/usr/bin/env python3
"""Build the local kageos plugin from canonical repository skills."""

from __future__ import annotations

import hashlib
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "skills"
TARGET = ROOT / "plugins" / "kageos" / "skills"
SKILLS = (
    "kageos",
    "kageos-contributor",
    "kageos-developer",
    "kageos-operator",
    "kageos-hub-publisher",
)


def assert_skill(path: Path) -> None:
    skill = path / "SKILL.md"
    if not skill.is_file():
        raise ValueError(f"missing SKILL.md: {path}")
    text = skill.read_text(encoding="utf-8")
    if not text.startswith("---\n") or "\ndescription:" not in text.split("---", 2)[1]:
        raise ValueError(f"invalid SKILL.md frontmatter: {skill}")


def sync() -> None:
    TARGET.mkdir(parents=True, exist_ok=True)
    for name in SKILLS:
        source = SOURCE / name
        destination = TARGET / name
        assert_skill(source)
        if destination.exists():
            shutil.rmtree(destination)
        shutil.copytree(source, destination, ignore=shutil.ignore_patterns("__pycache__", "*.pyc"))


def tree_hashes(root: Path) -> dict[str, str]:
    result = {}
    for path in sorted(root.rglob("*")):
        if not path.is_file() or "__pycache__" in path.parts or path.suffix == ".pyc":
            continue
        result[str(path.relative_to(root))] = hashlib.sha256(path.read_bytes()).hexdigest()
    return result


def compare() -> None:
    for name in SKILLS:
        if tree_hashes(SOURCE / name) != tree_hashes(TARGET / name):
            raise ValueError(f"packaged skill differs from source: {name}")


def main() -> int:
    sync()
    compare()
    print(f"packaged {len(SKILLS)} skills into {TARGET}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
