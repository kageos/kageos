import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
sys.path.insert(0, str(SCRIPT_DIR))

import render_report  # noqa: E402


def sample_report() -> dict:
    return {
        "schema_version": "kageos.operator-report.v1",
        "status": "verified",
        "directory": "/system/demos/meeting",
        "scenario": "创建、回读并清理一条测试记录",
        "started_at": "2026-08-11T09:00:00+08:00",
        "finished_at": "2026-08-11T09:01:00+08:00",
        "auth_mode": "access_token",
        "source_ref": "operator-test",
        "trace_id": "trace-test",
        "checks": [
            {
                "operation": "table.search",
                "full_code_path": "/system/demos/meeting/rooms.table",
                "status": "passed",
                "evidence": "测试记录已出现",
            }
        ],
        "automations": [],
        "cleanup": [{"operation": "table.delete", "status": "passed"}],
        "issues": [],
        "sensitive_fields": [],
    }


class RenderReportTest(unittest.TestCase):
    def test_load_report_rejects_non_contract_top_level_status(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "report.json"
            report = sample_report()
            report["status"] = "passed"
            path.write_text(json.dumps(report), encoding="utf-8")

            with self.assertRaisesRegex(ValueError, "verified or blocked"):
                render_report.load_report(path)

    def test_renderers_escape_untrusted_report_text(self) -> None:
        report = sample_report()
        report["directory"] = "/demo/[unsafe](javascript:alert(1))"
        report["scenario"] = "<script>alert(1)</script>\n# injected heading"

        markdown = render_report.render_markdown(report)
        rendered_html = render_report.render_html(report)

        self.assertNotIn("[unsafe](javascript:", markdown)
        self.assertNotIn("\n# injected heading", markdown)
        self.assertIn("&lt;script&gt;", markdown)
        self.assertNotIn("<script>", rendered_html)
        self.assertIn("&lt;script&gt;", rendered_html)

    def test_cli_writes_markdown_and_html_next_to_json(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            report_path = Path(directory) / "meeting.operator-report.json"
            report_path.write_text(json.dumps(sample_report(), ensure_ascii=False), encoding="utf-8")

            completed = subprocess.run(
                [sys.executable, str(SCRIPT_DIR / "render_report.py"), str(report_path)],
                check=True,
                capture_output=True,
                text=True,
            )

            result = json.loads(completed.stdout)
            markdown_path = report_path.with_suffix(".md")
            html_path = report_path.with_suffix(".html")
            self.assertEqual(result["status"], "verified")
            self.assertEqual(Path(result["outputs"]["markdown"]).resolve(), markdown_path.resolve())
            self.assertEqual(Path(result["outputs"]["html"]).resolve(), html_path.resolve())
            self.assertIn("Kageos 验证报告", markdown_path.read_text(encoding="utf-8"))
            self.assertIn("<!doctype html>", html_path.read_text(encoding="utf-8"))

    def test_format_and_output_directory_are_respected(self) -> None:
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            output_dir = root / "rendered"
            output_dir.mkdir()
            report_path = root / "report.json"
            report_path.write_text(json.dumps(sample_report()), encoding="utf-8")

            subprocess.run(
                [
                    sys.executable,
                    str(SCRIPT_DIR / "render_report.py"),
                    str(report_path),
                    "--format",
                    "md",
                    "--output-dir",
                    str(output_dir),
                ],
                check=True,
                capture_output=True,
                text=True,
            )

            self.assertTrue((output_dir / "report.md").is_file())
            self.assertFalse((output_dir / "report.html").exists())


if __name__ == "__main__":
    unittest.main()
