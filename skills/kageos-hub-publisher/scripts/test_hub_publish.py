import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

import hub_publish
from hub_publish import compose_submission, normalize_media_manifest, prepare_submission


class HubPublishMediaTest(unittest.TestCase):
    def test_manifest_resolves_files_and_defaults_images_into_description(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "result.png").write_bytes(b"png")
            manifest = root / "media.json"
            manifest.write_text(
                '{"items":[{"file":"result.png","alt":"执行结果","caption":"展示真实执行结果。"}]}',
                encoding="utf-8",
            )

            parsed = normalize_media_manifest(manifest)

            self.assertEqual(parsed["gallery_mode"], "replace")
            self.assertTrue(parsed["items"][0]["include_in_description"])
            self.assertEqual(parsed["items"][0]["file"], str((root / "result.png").resolve()))

    def test_compose_builds_gallery_and_escaped_inline_evidence(self):
        submission = {"description_html": "<p>原始介绍</p>"}
        manifest = {
            "gallery_mode": "replace",
            "append_to_description": True,
            "description_heading": "实际效果",
            "items": [{
                "alt": "趋势 <看板>",
                "caption": "展示 A & B 的结果。",
                "include_in_description": True,
            }],
        }
        assets = [{
            "url": "https://assets.hub.kageos.ai/assets/public/product-media/1/result.png",
            "content_type": "image/png",
        }]

        prepared = compose_submission(submission, manifest, assets)

        self.assertEqual(prepared["gallery"][0]["kind"], "image")
        self.assertIn("<p>原始介绍</p>", prepared["description_html"])
        self.assertIn("趋势 &lt;看板&gt;", prepared["description_html"])
        self.assertIn("A &amp; B", prepared["description_html"])

    def test_compose_appends_existing_gallery_and_deduplicates_urls(self):
        url = "https://assets.hub.kageos.ai/assets/public/product-media/1/demo.png"
        submission = {"gallery": [{"url": url, "kind": "image"}]}
        manifest = {
            "gallery_mode": "append",
            "append_to_description": False,
            "description_heading": "实际效果",
            "items": [{"alt": "结果", "caption": "结果图", "include_in_description": False}],
        }
        assets = [{"url": url, "content_type": "image/png"}]

        prepared = compose_submission(submission, manifest, assets)

        self.assertEqual(len(prepared["gallery"]), 1)

    def test_manifest_rejects_video_requested_for_inline_description(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "demo.mp4").write_bytes(b"video")
            manifest = root / "media.json"
            manifest.write_text(
                '{"items":[{"file":"demo.mp4","alt":"演示视频","caption":"完整操作。","include_in_description":true}]}',
                encoding="utf-8",
            )

            with self.assertRaisesRegex(ValueError, "cannot embed video"):
                normalize_media_manifest(manifest)

    def test_prepare_uploads_media_and_writes_ready_submission_without_submitting(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            screenshot = root / "result.png"
            screenshot.write_bytes(b"png")
            submission = root / "submission.json"
            submission.write_text('{"name":"巡检助手","description_html":"<p>介绍</p>"}', encoding="utf-8")
            manifest = root / "media.json"
            manifest.write_text(
                '{"items":[{"file":"result.png","alt":"巡检趋势","caption":"展示巡检结果。"}]}',
                encoding="utf-8",
            )
            output = root / "ready.json"
            completed = {
                "asset": {
                    "id": 9,
                    "url": "https://assets.hub.kageos.ai/assets/public/product-media/1/result.png",
                    "content_type": "image/png",
                }
            }

            with patch.object(hub_publish, "upload", return_value=completed) as mocked_upload:
                result = prepare_submission("https://hub.example/api/v1", "secret", submission, manifest, output)

            mocked_upload.assert_called_once_with(
                "https://hub.example/api/v1", "secret", str(screenshot.resolve()), "product-media", True
            )
            prepared = json.loads(output.read_text(encoding="utf-8"))
            self.assertEqual(result["uploaded"], 1)
            self.assertEqual(prepared["gallery"][0]["caption"], "展示巡检结果。")
            self.assertIn("<img", prepared["description_html"])

    def test_prepare_fails_preflight_before_uploading_too_many_gallery_items(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "result.png").write_bytes(b"png")
            existing = [{"url": f"https://assets.example/{index}.png", "kind": "image"} for index in range(8)]
            submission = root / "submission.json"
            submission.write_text(json.dumps({"gallery": existing}), encoding="utf-8")
            manifest = root / "media.json"
            manifest.write_text(
                '{"gallery_mode":"append","items":[{"file":"result.png","alt":"结果","caption":"结果图"}]}',
                encoding="utf-8",
            )

            with patch.object(hub_publish, "upload") as mocked_upload:
                with self.assertRaisesRegex(ValueError, "exceeds 8"):
                    prepare_submission("https://hub.example/api/v1", "secret", submission, manifest, root / "ready.json")

            mocked_upload.assert_not_called()


if __name__ == "__main__":
    unittest.main()
