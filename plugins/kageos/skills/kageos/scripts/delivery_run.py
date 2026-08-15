#!/usr/bin/env python3
"""Create and validate a credential-free kageos delivery evidence record."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import tempfile
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


SCHEMA_VERSION = "kageos.delivery-run.v1"
STAGES = (
    "design",
    "local_build",
    "platform_build",
    "operator_verify",
    "bundle",
    "publish_prepare",
    "publish_submit",
    "publish_status",
)
STATUSES = {"pending", "passed", "blocked"}
SENSITIVE = re.compile(r"(authorization|cookie|password|secret|token|signed[_-]?url)", re.I)


def now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def load_json(path: Path) -> dict[str, Any]:
    value = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(value, dict):
        raise ValueError(f"{path} must contain a JSON object")
    return value


def write_json(path: Path, value: dict[str, Any]) -> None:
    path = path.expanduser().resolve()
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            json.dump(value, handle, ensure_ascii=False, indent=2)
            handle.write("\n")
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def artifact(path: Path) -> dict[str, Any]:
    resolved = path.expanduser().resolve(strict=True)
    if not resolved.is_file():
        raise ValueError(f"artifact is not a file: {resolved}")
    digest = hashlib.sha256()
    with resolved.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return {
        "path": str(resolved),
        "size": resolved.stat().st_size,
        "sha256": digest.hexdigest(),
        "recorded_at": now(),
    }


def assert_no_sensitive(value: Any, trail: str = "root") -> None:
    if isinstance(value, dict):
        for key, child in value.items():
            if SENSITIVE.search(str(key)):
                raise ValueError(f"sensitive field is forbidden: {trail}.{key}")
            assert_no_sensitive(child, f"{trail}.{key}")
    elif isinstance(value, list):
        for index, child in enumerate(value):
            assert_no_sensitive(child, f"{trail}[{index}]")
    elif isinstance(value, str) and re.search(r"(?i)(bearer\s+\S+|x-token\s*[:=]|sk-[a-z0-9_-]{12,})", value):
        raise ValueError(f"credential-like value is forbidden: {trail}")


def parse_time(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def validate_artifact(entry: dict[str, Any]) -> None:
    path = Path(str(entry.get("path", "")))
    if not path.is_absolute() or not path.is_file():
        raise ValueError(f"artifact path is missing or not absolute: {path}")
    actual = artifact(path)
    if entry.get("sha256") != actual["sha256"] or entry.get("size") != actual["size"]:
        raise ValueError(f"artifact changed after recording: {path}")


def validate(run: dict[str, Any]) -> None:
    if run.get("schema_version") != SCHEMA_VERSION:
        raise ValueError(f"schema_version must be {SCHEMA_VERSION}")
    directory = run.get("directory")
    if not isinstance(directory, str) or not directory.startswith("/") or len(directory.split("/")) < 4:
        raise ValueError("directory must be a full_code_path such as /user/app/package")
    stages = run.get("stages")
    if not isinstance(stages, dict) or tuple(stages.keys()) != STAGES:
        raise ValueError("stages must contain the canonical ordered stage set")

    seen_non_passed = False
    for stage in STAGES:
        entry = stages[stage]
        if not isinstance(entry, dict) or entry.get("status") not in STATUSES:
            raise ValueError(f"invalid stage entry: {stage}")
        status = entry["status"]
        if status == "passed" and seen_non_passed:
            raise ValueError(f"stage passed before a prerequisite: {stage}")
        if status != "passed":
            seen_non_passed = True
        artifacts = entry.get("artifacts", [])
        if not isinstance(artifacts, list):
            raise ValueError(f"artifacts must be a list: {stage}")
        for item in artifacts:
            validate_artifact(item)

    operator = stages["operator_verify"]
    if operator["status"] == "passed":
        if not operator["artifacts"]:
            raise ValueError("operator_verify requires a report artifact")
        report = load_json(Path(operator["artifacts"][0]["path"]))
        if report.get("schema_version") != "kageos.operator-report.v1" or report.get("status") != "verified":
            raise ValueError("operator report must be kageos.operator-report.v1 with verified status")
        if report.get("directory") != directory:
            raise ValueError("operator report directory does not match delivery directory")
        built_at = stages["platform_build"].get("recorded_at")
        finished_at = report.get("finished_at")
        if built_at and finished_at and parse_time(finished_at) < parse_time(built_at):
            raise ValueError("operator report predates the latest platform build")

    bundle = stages["bundle"]
    if bundle["status"] == "passed":
        if not bundle["artifacts"]:
            raise ValueError("bundle requires a capability bundle artifact")
        bundle_json = load_json(Path(bundle["artifacts"][0]["path"]))
        if bundle_json.get("schema_version") != "capability.bundle.v1":
            raise ValueError("bundle artifact must use capability.bundle.v1")

    submitted = stages["publish_submit"]
    if submitted["status"] == "passed" and submitted.get("confirmed") is not True:
        raise ValueError("publish_submit passed without recorded confirmation")

    assert_no_sensitive(run)


def init_command(args: argparse.Namespace) -> None:
    timestamp = now()
    run = {
        "schema_version": SCHEMA_VERSION,
        "run_id": f"kageos-delivery-{uuid.uuid4().hex[:12]}",
        "directory": args.directory.rstrip("/"),
        "created_at": timestamp,
        "updated_at": timestamp,
        "stages": {
            stage: {"status": "pending", "recorded_at": "", "note": "", "artifacts": []}
            for stage in STAGES
        },
    }
    validate(run)
    write_json(args.output, run)
    print(args.output.expanduser().resolve())


def record_command(args: argparse.Namespace) -> None:
    path = args.run.expanduser().resolve(strict=True)
    run = load_json(path)
    if args.stage == "publish_submit" and args.status == "passed" and not args.confirmed:
        raise ValueError("publish_submit requires --confirmed")
    index = STAGES.index(args.stage)
    if args.status == "passed":
        missing = [name for name in STAGES[:index] if run["stages"][name]["status"] != "passed"]
        if missing:
            raise ValueError(f"prerequisite stages have not passed: {', '.join(missing)}")
    entry = run["stages"][args.stage]
    entry["status"] = args.status
    entry["recorded_at"] = now()
    entry["note"] = args.note or ""
    entry["artifacts"] = [artifact(item) for item in args.artifact]
    if args.stage == "publish_submit":
        entry["confirmed"] = bool(args.confirmed)
    run["updated_at"] = now()
    validate(run)
    write_json(path, run)
    print(path)


def validate_command(args: argparse.Namespace) -> None:
    run = load_json(args.run.expanduser().resolve(strict=True))
    validate(run)
    print("valid")


def reset_command(args: argparse.Namespace) -> None:
    path = args.run.expanduser().resolve(strict=True)
    run = load_json(path)
    validate(run)
    start = STAGES.index(args.from_stage)
    for stage in STAGES[start:]:
        run["stages"][stage] = {
            "status": "pending",
            "recorded_at": "",
            "note": args.reason if stage == args.from_stage else "",
            "artifacts": [],
        }
    run["updated_at"] = now()
    validate(run)
    write_json(path, run)
    print(path)


def show_command(args: argparse.Namespace) -> None:
    run = load_json(args.run.expanduser().resolve(strict=True))
    validate(run)
    print(f"{run['directory']}  {run['run_id']}")
    for stage in STAGES:
        entry = run["stages"][stage]
        suffix = f"  {entry['note']}" if entry.get("note") else ""
        print(f"{stage:18} {entry['status']}{suffix}")


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser(description=__doc__)
    commands = root.add_subparsers(dest="command", required=True)
    init = commands.add_parser("init")
    init.add_argument("--output", type=Path, required=True)
    init.add_argument("--directory", required=True)
    init.set_defaults(func=init_command)
    record = commands.add_parser("record")
    record.add_argument("--run", type=Path, required=True)
    record.add_argument("--stage", choices=STAGES, required=True)
    record.add_argument("--status", choices=("passed", "blocked"), required=True)
    record.add_argument("--note")
    record.add_argument("--artifact", type=Path, action="append", default=[])
    record.add_argument("--confirmed", action="store_true")
    record.set_defaults(func=record_command)
    check = commands.add_parser("validate")
    check.add_argument("--run", type=Path, required=True)
    check.set_defaults(func=validate_command)
    reset = commands.add_parser("reset")
    reset.add_argument("--run", type=Path, required=True)
    reset.add_argument("--from-stage", choices=STAGES, required=True)
    reset.add_argument("--reason", required=True)
    reset.set_defaults(func=reset_command)
    show = commands.add_parser("show")
    show.add_argument("--run", type=Path, required=True)
    show.set_defaults(func=show_command)
    return root


def main() -> int:
    args = parser().parse_args()
    try:
        args.func(args)
    except (OSError, ValueError, json.JSONDecodeError) as error:
        raise SystemExit(f"error: {error}") from error
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
