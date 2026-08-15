import importlib.util
import json
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("delivery_run.py")
SPEC = importlib.util.spec_from_file_location("delivery_run", SCRIPT)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(MODULE)


class DeliveryRunTest(unittest.TestCase):
    def new_run(self, root: Path):
        timestamp = MODULE.now()
        return {
            "schema_version": MODULE.SCHEMA_VERSION,
            "run_id": "kageos-delivery-test",
            "directory": "/user/app/package",
            "created_at": timestamp,
            "updated_at": timestamp,
            "stages": {
                stage: {"status": "pending", "recorded_at": "", "note": "", "artifacts": []}
                for stage in MODULE.STAGES
            },
        }

    def test_empty_run_is_valid(self):
        with tempfile.TemporaryDirectory() as directory:
            MODULE.validate(self.new_run(Path(directory)))

    def test_stage_order_is_enforced(self):
        with tempfile.TemporaryDirectory() as directory:
            run = self.new_run(Path(directory))
            run["stages"]["local_build"]["status"] = "passed"
            with self.assertRaisesRegex(ValueError, "before a prerequisite"):
                MODULE.validate(run)

    def test_verified_report_and_bundle_are_accepted(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            run = self.new_run(root)
            report_path = root / "operator.json"
            report_path.write_text(json.dumps({
                "schema_version": "kageos.operator-report.v1",
                "status": "verified",
                "directory": run["directory"],
                "finished_at": MODULE.now(),
            }))
            bundle_path = root / "bundle.json"
            bundle_path.write_text(json.dumps({"schema_version": "capability.bundle.v1"}))
            for stage in MODULE.STAGES[:4]:
                run["stages"][stage]["status"] = "passed"
                run["stages"][stage]["recorded_at"] = "2026-01-01T00:00:00Z"
            run["stages"]["operator_verify"]["artifacts"] = [MODULE.artifact(report_path)]
            run["stages"]["bundle"]["status"] = "passed"
            run["stages"]["bundle"]["recorded_at"] = MODULE.now()
            run["stages"]["bundle"]["artifacts"] = [MODULE.artifact(bundle_path)]
            MODULE.validate(run)

    def test_credentials_are_rejected(self):
        with tempfile.TemporaryDirectory() as directory:
            run = self.new_run(Path(directory))
            run["api_token"] = "secret"
            with self.assertRaisesRegex(ValueError, "sensitive field"):
                MODULE.validate(run)

    def test_submit_requires_persisted_confirmation(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            run = self.new_run(root)
            report_path = root / "operator.json"
            report_path.write_text(json.dumps({
                "schema_version": "kageos.operator-report.v1",
                "status": "verified",
                "directory": run["directory"],
                "finished_at": "2026-01-02T00:00:00Z",
            }))
            bundle_path = root / "bundle.json"
            bundle_path.write_text(json.dumps({"schema_version": "capability.bundle.v1"}))
            for stage in MODULE.STAGES:
                run["stages"][stage]["status"] = "passed"
                run["stages"][stage]["recorded_at"] = "2026-01-01T00:00:00Z"
            run["stages"]["operator_verify"]["artifacts"] = [MODULE.artifact(report_path)]
            run["stages"]["bundle"]["artifacts"] = [MODULE.artifact(bundle_path)]
            with self.assertRaisesRegex(ValueError, "without recorded confirmation"):
                MODULE.validate(run)
            run["stages"]["publish_submit"]["confirmed"] = True
            MODULE.validate(run)


if __name__ == "__main__":
    unittest.main()
