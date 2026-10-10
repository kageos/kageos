#!/usr/bin/env python3
"""Regression checks for isolated cache loading and fail-fast infrastructure boot."""
import pathlib
import re
import subprocess
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
COMMON = (ROOT / 'deploy/prod/entrypoint-common.sh').read_text().replace('cd /app', ':')
AIO = (ROOT / 'deploy/aio/entrypoint-aio.sh').read_text().replace('source /app/entrypoint-common.sh', ':').rsplit('main "$@"', 1)[0]

MAIN = (ROOT / 'deploy/prod/entrypoint-main.sh').read_text()
CORE_WAIT = re.search(r'^wait_core_ready\(\) \{\n.*?^\}', MAIN, re.M | re.S).group(0)
SHUTDOWN = re.search(r'^shutdown\(\) \{\n.*?^\}', MAIN, re.M | re.S).group(0)

class BootTests(unittest.TestCase):
    def run_shell(self, script):
        result = subprocess.run(['bash', '-c', COMMON + '\n' + AIO + '\ntimeout() { shift; \"$@\"; }\nexport_defaults\nMYSQL_ROOT_PASSWORD=test MINIO_ROOT_PASSWORD=test\n' + script], text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        return result.stdout

    def test_local_image_skips_network(self):
        self.run_shell('''
podman() { [[ "$*" == 'image exists cached' ]]; }
kageos_pull_with_progress() { echo unexpected-pull; return 99; }
kageos_ensure_image cached
''')

    def test_existing_container_skips_new_image_download(self):
        self.run_shell('''
podman() { [[ "$*" == 'container exists existing-minio' ]]; }
kageos_ensure_image() { exit 98; }
prepare_infra_image existing-minio retired-reference
''')

    def test_upstream_cache_alias_skips_download(self):
        self.run_shell("""
tagged=0
podman() {
  case "$1 $2" in
    'image exists')
      [[ "$3" == docker.io/library/mysql:8.0.45 || "$tagged" == 1 ]] ;;
    'tag docker.io/library/mysql:8.0.45') tagged=1 ;;
    *) return 99 ;;
  esac
}
kageos_pull_with_progress() { exit 98; }
kageos_ensure_image m.daocloud.io/docker.io/library/mysql:8.0.45
[[ "$tagged" == 1 ]]
""")

    def test_archive_loads_without_pull(self):
        with tempfile.TemporaryDirectory() as folder:
            self.run_shell('''
KAGEOS_AIO_IMAGE_CACHE_DIR=''' + folder + '''
image=example/mysql:8
key=$(printf '%s' "$image" | sha256sum | awk '{print $1}')
printf archive > "$KAGEOS_AIO_IMAGE_CACHE_DIR/$key.tar"
loaded=0
podman() {
  case "$1 $2" in
    'image exists') [[ "$loaded" == 1 ]] ;;
    'load -i') loaded=1 ;;
    *) return 99 ;;
  esac
}
kageos_pull_with_progress() { return 99; }
kageos_ensure_image "$image"
[[ "$loaded" == 1 ]]
''')

    def test_pull_failure_never_waits_or_runs(self):
        for service in ('mysql', 'nats', 'minio'):
            with self.subTest(service=service):
                self.run_shell('''
podman_container_running() { return 1; }
podman() { [[ "$1" != run ]] || exit 98; return 1; }
kageos_ensure_image() { return 23; }
wait_tcp() { exit 97; }
if kageos_run_timed_stage test start_''' + service + '''; then exit 96; else [[ "$?" == 23 ]]; fi
''')

    def test_run_failure_never_waits(self):
        for service in ('mysql', 'nats', 'minio'):
            with self.subTest(service=service):
                self.run_shell('''
podman_container_running() { return 1; }
podman() { [[ "$1" == run ]] && return 24; return 1; }
kageos_ensure_image() { return 0; }
wait_tcp() { exit 97; }
if kageos_run_timed_stage test start_''' + service + '''; then exit 96; else [[ "$?" == 24 ]]; fi
''')

    def test_exited_container_fails_without_sleep(self):
        self.run_shell("""
podman_container_running() { return 1; }
infra_diagnostics() { echo diagnostics; }
sleep() { exit 98; }
probe() { exit 97; }
if wait_service_ready dead test 30 probe; then exit 96; fi
""")

    def test_readiness_timeout_and_delayed_success(self):
        self.run_shell("""
ensure_container_running() { return 0; }
infra_diagnostics() { echo diagnostics; }
SECONDS=0
sleep() { SECONDS=$((SECONDS + 2)); }
probe() { return 1; }
if wait_service_ready slow test 4 probe; then exit 98; fi
[[ "$SECONDS" -ge 4 ]]
SECONDS=0
probe() { [[ "$SECONDS" -ge 2 ]]; }
wait_service_ready slow test 10 probe
[[ "$SECONDS" -lt 10 ]]
""")

    def test_invalid_timeout_does_not_probe(self):
        self.run_shell("""
probe() { exit 99; }
if wait_service_ready test test invalid probe; then exit 98; fi
""")

    def test_diagnostics_redacts_credentials(self):
        output = self.run_shell("""
MYSQL_ROOT_PASSWORD=test-secret
MINIO_ROOT_PASSWORD=minio-secret
NATS_PASSWORD=nats-secret
podman() {
  [[ "$1" != logs ]] || echo 'test-secret minio-secret nats-secret'
  return 0
}
infra_diagnostics test test 2>&1
""")
        self.assertNotIn('test-secret', output)
        self.assertNotIn('minio-secret', output)
        self.assertNotIn('nats-secret', output)
        self.assertIn('[REDACTED]', output)

    def test_core_exit_fails_before_network_probe(self):
        self.run_shell(CORE_WAIT + """
CORE_PID=999999
kill() { return 1; }
curl() { exit 98; }
sleep() { exit 97; }
if wait_core_ready; then exit 96; fi
""")

    def test_core_timeout_uses_elapsed_time(self):
        self.run_shell(CORE_WAIT + """
CORE_PID=999999
KAGEOS_AIO_CORE_READY_TIMEOUT=4
SECONDS=0
kill() { return 0; }
curl() { SECONDS=$((SECONDS + 3)); return 1; }
sleep() { SECONDS=$((SECONDS + 2)); }
if wait_core_ready; then exit 98; fi
[[ "$SECONDS" == 5 ]]
""")

    def test_shutdown_preserves_failure_status(self):
        output = self.run_shell(SHUTDOWN + """
CORE_PID=101 APP_BASE_PID=102 PODMAN_PID=103
kill() { echo "stop-$2"; }
wait() { return 0; }
nginx() { echo nginx-stop; }
if (shutdown 7); then exit 98; else [[ "$?" == 7 ]]; fi
""")
        for marker in ('stop-101', 'stop-102', 'stop-103', 'nginx-stop'):
            self.assertIn(marker, output)

    def test_progress_preserves_pull_status(self):
        self.run_shell('''
script() { [[ "$1 $2 $3" == '-q -e -c' ]] || exit 98; return 25; }
if kageos_pull_with_progress 'example/image:1'; then exit 97; else [[ "$?" == 25 ]]; fi
''')

if __name__ == '__main__':
    unittest.main()
