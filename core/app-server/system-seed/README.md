# System Capability Seeds

Place built-in `capability.bundle.v1` JSON files here.

Convention:

- `system/tools/*.json` installs into `/system/tools`
- `system/tools/openapi/*.json` installs into `/system/tools/openapi`

The app-server scans this directory on startup, but installs seed bundles only
while the target system app has no published version yet. The first deployment
uses `overwrite=true` and `force_diff=true` so SDK API diff can build metadata
from registered routes; later restarts skip these bundles to avoid rewriting
files and recompiling built-in system apps.
