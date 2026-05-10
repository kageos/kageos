# Official Capability Seeds

Place built-in `capability.bundle.v1` JSON files here.

Convention:

- `system/tools/*.json` installs into `/system/tools`
- `system/openapi/*.json` installs into `/system/openapi`
- `system/tools/openapi/*.json` installs into `/system/tools/openapi`

The app-server scans this directory on startup and installs each JSON with
`overwrite=true` and `force_diff=true`, so SDK API diff can rebuild from
registered routes.
