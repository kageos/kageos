#!/usr/bin/env python3
"""Optional local integration test; uses cached images and disposable containers."""
import os
import datetime
import hashlib
import hmac
import urllib.request
import pathlib
import runpy
import shlex
import subprocess
import tempfile
import uuid

ROOT = pathlib.Path(__file__).resolve().parents[1]
context = runpy.run_path(str(ROOT / 'scripts/test-aio-image-cache.py'))
def run(*args):
    return subprocess.check_output(args, text=True).strip()

suffix = uuid.uuid4().hex[:10]
names = {service: f'kageos-boot-test-{service}-{suffix}' for service in ('mysql', 'nats', 'minio')}
images = {
    'mysql': os.environ.get('TEST_MYSQL_IMAGE', 'docker.io/library/mysql:8.0'),
    'nats': os.environ.get('TEST_NATS_IMAGE', 'docker.io/library/nats:2.10.29-alpine'),
    'minio': os.environ.get('TEST_MINIO_IMAGE', 'm.daocloud.io/docker.io/bitnamilegacy/minio:2025.7.23-debian-12-r5'),
}
previous_minio = os.environ.get("TEST_MINIO_PREVIOUS_IMAGE", images["minio"])
for image in list(images.values()) + [previous_minio]:
    run('podman', 'image', 'exists', image)  # never download during this check

def s3_request(port, method, path, body=b''):
    now = datetime.datetime.now(datetime.timezone.utc)
    timestamp, day = now.strftime('%Y%m%dT%H%M%SZ'), now.strftime('%Y%m%d')
    payload_hash = hashlib.sha256(body).hexdigest()
    host = f'127.0.0.1:{port}'
    signed = 'host;x-amz-content-sha256;x-amz-date'
    headers = f'host:{host}\nx-amz-content-sha256:{payload_hash}\nx-amz-date:{timestamp}\n'
    canonical = '\n'.join([method, path, '', headers, signed, payload_hash])
    scope = f'{day}/us-east-1/s3/aws4_request'
    to_sign = '\n'.join(['AWS4-HMAC-SHA256', timestamp, scope, hashlib.sha256(canonical.encode()).hexdigest()])
    key = b'AWS4synthetic-boot-test'
    for part in (day, 'us-east-1', 's3', 'aws4_request'):
        key = hmac.new(key, part.encode(), hashlib.sha256).digest()
    signature = hmac.new(key, to_sign.encode(), hashlib.sha256).hexdigest()
    request = urllib.request.Request(f'http://{host}{path}', data=body if method == 'PUT' else None, method=method,
        headers={'X-Amz-Date': timestamp, 'X-Amz-Content-SHA256': payload_hash,
                 'Authorization': f'AWS4-HMAC-SHA256 Credential=test-user/{scope}, SignedHeaders={signed}, Signature={signature}'})
    with urllib.request.urlopen(request, timeout=10) as response:
        return response.read()

with tempfile.TemporaryDirectory(prefix='kageos-boot-test-') as folder:
    data_dir = pathlib.Path(folder) / 'minio-data'
    data_dir.mkdir()
    def start_minio(image):
        run('podman', 'run', '-d', '--pull=never', '--name', names['minio'],
            '--entrypoint', 'minio', '--user', '0', '-v', f'{data_dir}:/data',
            '-p', '127.0.0.1::19000', '-e', 'MINIO_ROOT_USER=test-user',
            '-e', 'MINIO_ROOT_PASSWORD=synthetic-boot-test', image,
            'server', '/data', '--address', ':19000')
        return run('podman', 'port', names['minio'], '19000/tcp').rsplit(':', 1)[1]
    try:
        # All three start before any readiness wait, as in the AIO entrypoint.
        run('podman', 'run', '-d', '--pull=never', '--name', names['mysql'],
            '-e', 'MYSQL_ROOT_PASSWORD=synthetic-boot-test', images['mysql'], '--port=13306')
        run('podman', 'run', '-d', '--pull=never', '--name', names['nats'],
            '-p', '127.0.0.1::14222', images['nats'], '--port', '14222',
            '--user', 'test', '--pass', 'synthetic-boot-test')
        minio_port = start_minio(previous_minio)
        nats_port = run('podman', 'port', names['nats'], '14222/tcp').rsplit(':', 1)[1]
        script = context['COMMON'] + '\n' + context['AIO'] + '\n' + '''
# macOS may lack GNU timeout; this test shim preserves the probe's environment.
timeout() {
  python3 -c 'import subprocess, sys
try:
    sys.exit(subprocess.run(sys.argv[2:], timeout=float(sys.argv[1])).returncode)
except subprocess.TimeoutExpired:
    sys.exit(124)' "$@"
}
export_defaults
MYSQL_ROOT_PASSWORD=synthetic-boot-test
MINIO_ROOT_PASSWORD=synthetic-boot-test
NATS_USER=test
NATS_PASSWORD=synthetic-boot-test
KAGEOS_AIO_MYSQL_READY_TIMEOUT=120
KAGEOS_AIO_NATS_READY_TIMEOUT=30
KAGEOS_AIO_MINIO_READY_TIMEOUT=30
'''
        for service, name in names.items():
            script += f'{service.upper()}_CONTAINER_NAME={shlex.quote(name)}\n'
        script += f'NATS_PORT={nats_port}\nMINIO_PORT={minio_port}\n'
        script += f'AIO_INFRA_DIR={shlex.quote(folder)}\n'
        script += '''
write_infra_files
wait_nats_ready
NATS_PASSWORD=wrong-synthetic-password
if nats_probe; then echo 'ERROR: wrong NATS password accepted'; exit 1; fi
NATS_PASSWORD=synthetic-boot-test
wait_minio_ready
wait_mysql_ready
# Verify the init SQL created the expected database, not just a listening port.
timeout 5 podman exec -e "MYSQL_PWD=$MYSQL_ROOT_PASSWORD" "$MYSQL_CONTAINER_NAME" \\
  mysql --protocol=TCP -h 127.0.0.1 -P 13306 -uroot -e 'USE `app-server`; SELECT 1' >/dev/null
podman stop -t 1 "$NATS_CONTAINER_NAME" >/dev/null
if wait_nats_ready; then echo 'ERROR: stopped NATS reported ready'; exit 1; fi
echo 'Disposable MySQL / NATS / MinIO integration checks passed'
'''
        subprocess.run(['bash', '-c', script], check=True, timeout=180)
        s3_request(minio_port, 'PUT', '/boot-test-bucket')
        marker = b'kageos-release-persistence-test'
        s3_request(minio_port, 'PUT', '/boot-test-bucket/marker.txt', marker)
        assert s3_request(minio_port, 'GET', '/boot-test-bucket/marker.txt') == marker
        run('podman', 'stop', '-t', '1', names['minio'])
        run('podman', 'rm', names['minio'])
        minio_port = start_minio(images['minio'])
        # Reuse the real probes and persisted data with the replacement packaging.
        restart_probe = script[:script.index('write_infra_files\nwait_nats_ready')]
        restart_probe += f'MINIO_PORT={minio_port}\nwait_minio_ready\n'
        subprocess.run(['bash', '-c', restart_probe], check=True, timeout=60)
        assert s3_request(minio_port, 'GET', '/boot-test-bucket/marker.txt') == marker
        print('S3 upload/download and persisted-data restart/upgrade checks passed', flush=True)
    finally:
        # Delete only the exact disposable test containers and this temporary test data.
        for name in names.values():
            subprocess.run(['podman', 'rm', '-f', name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
