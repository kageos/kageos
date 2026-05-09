# Official Directory Seeds

Place built-in directory bundle JSON files here.

Convention:

- `tools/*.json` imports into `/system/tools`
- `openapi/*.json` imports into `/system/openapi`
- `official/*.json` imports into `/system/official`

The app-server scans this directory on startup and imports each JSON with
`skip_if_exists`.
