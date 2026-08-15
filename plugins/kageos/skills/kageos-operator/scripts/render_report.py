#!/usr/bin/env python3
"""Render a kageos operator JSON report as readable Markdown and HTML."""

from __future__ import annotations

import argparse
import html
import json
from pathlib import Path
from typing import Any


SCHEMA_VERSION = "kageos.operator-report.v1"


def text(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, (dict, list)):
        return json.dumps(value, ensure_ascii=False, sort_keys=True)
    return str(value)


def markdown_cell(value: Any) -> str:
    return markdown_text(value)


def markdown_text(value: Any) -> str:
    escaped = html.escape(text(value), quote=False)
    for character, entity in {
        "\\": "&#92;",
        "`": "&#96;",
        "*": "&#42;",
        "_": "&#95;",
        "[": "&#91;",
        "]": "&#93;",
        "#": "&#35;",
        "!": "&#33;",
        "|": "&#124;",
    }.items():
        escaped = escaped.replace(character, entity)
    return escaped.replace("\r\n", "\n").replace("\r", "\n").replace("\n", "<br>")


def section(report: dict[str, Any], key: str) -> list[dict[str, Any]]:
    value = report.get(key, [])
    if value is None:
        return []
    if not isinstance(value, list) or any(not isinstance(item, dict) for item in value):
        raise ValueError(f"report field {key!r} must be an array of objects")
    return value


def string_list(report: dict[str, Any], key: str) -> list[str]:
    value = report.get(key, [])
    if value is None:
        return []
    if not isinstance(value, list):
        raise ValueError(f"report field {key!r} must be an array")
    return [text(item) for item in value if text(item)]


def load_report(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        raise ValueError(f"invalid JSON: {error}") from error
    if not isinstance(value, dict):
        raise ValueError("report root must be a JSON object")
    if value.get("schema_version") != SCHEMA_VERSION:
        raise ValueError(f"schema_version must be {SCHEMA_VERSION}")
    if value.get("status") not in {"verified", "blocked"}:
        raise ValueError("status must be verified or blocked")
    for key in ("checks", "automations", "cleanup"):
        section(value, key)
    for key in ("issues", "retained_evidence", "sensitive_fields", "synthetic_markers"):
        string_list(value, key)
    return value


def render_markdown(report: dict[str, Any]) -> str:
    status = text(report.get("status")).upper()
    directory = text(report.get("directory")) or "未指定目录"
    lines = [
        f"# kageos 验证报告：{markdown_text(directory)}",
        "",
        f"**状态：{status}**",
        "",
    ]
    scenario = text(report.get("scenario"))
    if scenario:
        lines.extend([markdown_text(scenario), ""])

    metadata = [
        ("目录", directory),
        ("目标", report.get("target")),
        ("认证模式", report.get("auth_mode")),
        ("来源标识", report.get("source_ref")),
        ("Trace ID", report.get("trace_id")),
        ("开始时间", report.get("started_at")),
        ("完成时间", report.get("finished_at")),
    ]
    lines.extend(["## 运行信息", "", "| 项目 | 内容 |", "| --- | --- |"])
    lines.extend(f"| {name} | {markdown_cell(value) or '—'} |" for name, value in metadata)
    lines.append("")

    append_markdown_table(lines, "验证检查", section(report, "checks"), [
        ("操作", "operation"), ("路径", "full_code_path"), ("状态", "status"), ("证据", "evidence")
    ])
    append_markdown_table(lines, "自动化证据", section(report, "automations"), [
        ("代码", "code"), ("类型", "kind"), ("状态", "status"), ("证据或原因", "evidence", "reason")
    ])
    append_markdown_table(lines, "清理结果", section(report, "cleanup"), [
        ("资源或操作", "resource", "operation"), ("路径", "full_code_path"), ("状态", "status"), ("证据", "evidence")
    ])

    append_markdown_list(lines, "保留证据", string_list(report, "retained_evidence"))
    append_markdown_list(lines, "问题与注意事项", string_list(report, "issues"))
    append_markdown_list(lines, "敏感字段（仅字段名）", string_list(report, "sensitive_fields"))
    append_markdown_list(lines, "测试标记", string_list(report, "synthetic_markers"))
    lines.extend(["---", "", "由 `kageos-operator/scripts/render_report.py` 从机器可校验 JSON 生成。", ""])
    return "\n".join(lines)


def append_markdown_table(
    lines: list[str],
    title: str,
    rows: list[dict[str, Any]],
    columns: list[tuple[str, ...]],
) -> None:
    lines.extend([f"## {title}", ""])
    if not rows:
        lines.extend(["无。", ""])
        return
    lines.append("| " + " | ".join(column[0] for column in columns) + " |")
    lines.append("| " + " | ".join("---" for _ in columns) + " |")
    for row in rows:
        values = []
        for column in columns:
            value = next((row.get(key) for key in column[1:] if row.get(key) not in (None, "")), "")
            values.append(markdown_cell(value) or "—")
        lines.append("| " + " | ".join(values) + " |")
    lines.append("")


def append_markdown_list(lines: list[str], title: str, items: list[str]) -> None:
    if not items:
        return
    lines.extend([f"## {title}", ""])
    lines.extend(f"- {markdown_text(item).replace(chr(10), ' ')}" for item in items)
    lines.append("")


def h(value: Any) -> str:
    return html.escape(text(value), quote=True)


def html_table(rows: list[dict[str, Any]], columns: list[tuple[str, ...]]) -> str:
    if not rows:
        return '<p class="empty">无</p>'
    head = "".join(f"<th>{h(column[0])}</th>" for column in columns)
    body = []
    for row in rows:
        cells = []
        for column in columns:
            value = next((row.get(key) for key in column[1:] if row.get(key) not in (None, "")), "")
            cell = h(value) or "—"
            if column[0] == "状态":
                css_status = h(text(value).lower())
                cell = f'<span class="status status-{css_status}">{cell}</span>'
            cells.append(f"<td>{cell}</td>")
        body.append("<tr>" + "".join(cells) + "</tr>")
    return f'<div class="table-wrap"><table><thead><tr>{head}</tr></thead><tbody>{"".join(body)}</tbody></table></div>'


def html_list(items: list[str]) -> str:
    return "<ul>" + "".join(f"<li>{h(item)}</li>" for item in items) + "</ul>" if items else '<p class="empty">无</p>'


def render_html(report: dict[str, Any]) -> str:
    status = text(report.get("status")).lower()
    directory = text(report.get("directory")) or "未指定目录"
    checks = section(report, "checks")
    automations = section(report, "automations")
    cleanup = section(report, "cleanup")
    passed = sum(1 for item in checks if item.get("status") == "passed")
    metadata = [
        ("认证模式", report.get("auth_mode")),
        ("来源标识", report.get("source_ref")),
        ("Trace ID", report.get("trace_id")),
        ("完成时间", report.get("finished_at")),
    ]
    cards = "".join(
        f'<div class="meta-card"><span>{h(name)}</span><strong>{h(value) or "—"}</strong></div>'
        for name, value in metadata
    )
    return f"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>kageos 验证报告 · {h(directory)}</title>
<style>
:root{{--bg:#f4f7fb;--panel:#fff;--ink:#172033;--muted:#667085;--line:#e4e9f2;--brand:#2457a6;--ok:#16825d;--bad:#c33d52;--warn:#b7791f}}
*{{box-sizing:border-box}} body{{margin:0;background:linear-gradient(145deg,#edf4ff 0,#f7f9fc 42%,#f4f7fb 100%);color:var(--ink);font:15px/1.65 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif}}
.page{{max-width:1180px;margin:0 auto;padding:48px 24px 72px}} .hero{{background:linear-gradient(135deg,#173c75,#2f6fc7);color:#fff;border-radius:22px;padding:34px 38px;box-shadow:0 18px 45px rgba(27,67,124,.18)}}
.eyebrow{{font-size:12px;letter-spacing:.16em;text-transform:uppercase;opacity:.72}} h1{{margin:8px 0 10px;font-size:32px;line-height:1.25}} .scenario{{max-width:850px;margin:0;opacity:.9}} .hero-status{{display:inline-flex;margin-top:20px;padding:6px 12px;border-radius:999px;background:rgba(255,255,255,.16);font-weight:750;letter-spacing:.04em}}
.stats{{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin:18px 0}} .stat{{background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:18px 20px}} .stat strong{{display:block;font-size:27px}} .stat span,.meta-card span{{color:var(--muted);font-size:13px}}
.metadata{{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px;margin:18px 0}} .meta-card{{background:rgba(255,255,255,.86);border:1px solid var(--line);border-radius:14px;padding:14px 16px;min-width:0}} .meta-card strong{{display:block;overflow-wrap:anywhere}}
section{{background:var(--panel);border:1px solid var(--line);border-radius:18px;padding:24px;margin-top:18px;box-shadow:0 8px 24px rgba(31,54,87,.05)}} h2{{font-size:19px;margin:0 0 16px}} .table-wrap{{overflow:auto;border:1px solid var(--line);border-radius:12px}} table{{width:100%;border-collapse:collapse;min-width:720px}} th,td{{padding:12px 14px;text-align:left;vertical-align:top;border-bottom:1px solid var(--line)}} th{{background:#f7f9fc;font-size:13px;color:var(--muted)}} tr:last-child td{{border-bottom:0}} td{{overflow-wrap:anywhere}}
.status{{display:inline-flex;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:750;background:#eef2f7}} .status-passed,.status-verified{{background:#e8f7f1;color:var(--ok)}} .status-blocked,.status-failed{{background:#fff0f2;color:var(--bad)}} .empty{{color:var(--muted);margin:0}} ul{{margin:0;padding-left:20px}} li+li{{margin-top:7px}} footer{{color:var(--muted);text-align:center;margin-top:26px;font-size:13px}}
@media(max-width:720px){{.page{{padding:20px 14px 48px}}.hero{{padding:26px 22px}}h1{{font-size:26px}}.stats{{grid-template-columns:1fr}}.metadata{{grid-template-columns:1fr}}}}
</style>
</head>
<body><main class="page">
<header class="hero"><div class="eyebrow">kageos Operator Report</div><h1>{h(directory)}</h1><p class="scenario">{h(report.get("scenario")) or "目录运行验证结果"}</p><div class="hero-status">{h(status.upper())}</div></header>
<div class="stats"><div class="stat"><strong>{len(checks)}</strong><span>验证检查</span></div><div class="stat"><strong>{passed}</strong><span>通过检查</span></div><div class="stat"><strong>{len(cleanup)}</strong><span>清理项</span></div></div>
<div class="metadata">{cards}</div>
<section><h2>验证检查</h2>{html_table(checks, [("操作","operation"),("路径","full_code_path"),("状态","status"),("证据","evidence")])}</section>
<section><h2>自动化证据</h2>{html_table(automations, [("代码","code"),("类型","kind"),("状态","status"),("证据或原因","evidence","reason")])}</section>
<section><h2>清理结果</h2>{html_table(cleanup, [("资源或操作","resource","operation"),("路径","full_code_path"),("状态","status"),("证据","evidence")])}</section>
<section><h2>保留证据</h2>{html_list(string_list(report,"retained_evidence"))}</section>
<section><h2>问题与注意事项</h2>{html_list(string_list(report,"issues"))}</section>
<section><h2>敏感字段（仅字段名）</h2>{html_list(string_list(report,"sensitive_fields"))}</section>
<footer>由 kageos-operator 从机器可校验 JSON 安全生成 · 不执行报告内 HTML</footer>
</main></body></html>"""


def output_path(report_path: Path, output_dir: Path, suffix: str) -> Path:
    name = report_path.name[:-5] if report_path.name.lower().endswith(".json") else report_path.stem
    return output_dir / f"{name}{suffix}"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path, help="kageos.operator-report.v1 JSON file")
    parser.add_argument("--format", choices=("both", "md", "html"), default="both")
    parser.add_argument("--output-dir", type=Path, help="output directory; defaults to the report directory")
    args = parser.parse_args()

    report_path = args.report.expanduser().resolve()
    report = load_report(report_path)
    output_dir = (args.output_dir or report_path.parent).expanduser().resolve()
    if not output_dir.is_dir():
        raise SystemExit(f"output directory does not exist: {output_dir}")

    outputs: dict[str, str] = {}
    if args.format in {"both", "md"}:
        path = output_path(report_path, output_dir, ".md")
        path.write_text(render_markdown(report), encoding="utf-8")
        outputs["markdown"] = str(path)
    if args.format in {"both", "html"}:
        path = output_path(report_path, output_dir, ".html")
        path.write_text(render_html(report), encoding="utf-8")
        outputs["html"] = str(path)
    print(json.dumps({"status": report["status"], "outputs": outputs}, ensure_ascii=False))


if __name__ == "__main__":
    main()
