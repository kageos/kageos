# Third-Party Licenses

This file is a maintained entry point for third-party license review. It is not
a legal opinion or a complete SBOM. Before a public release, regenerate and
review dependency inventories from the current lock files and container images.

## Source Dependencies

- Go modules are declared in `go.mod` and pinned in `go.sum`.
- Frontend packages are declared in `web/package.json` and pinned in
  `web/package-lock.json`.
- Generated application workspaces under `namespace/` are runtime data and are
  not part of the source distribution.

Useful review commands:

```bash
go list -m -json all
cd web && npm ls --all
```

## Runtime Images

The app base image installs command-line tools and Python packages for document,
image, audio/video, archive, OCR, and data-processing workloads. These tools are
invoked as external programs by generated applications; their upstream licenses
still apply to redistribution and runtime use.

The current image documentation lists notable tools in
`deploy/base/images/app-base/README.md`, including FFmpeg, Ghostscript,
Poppler-utils, GraphicsMagick, ImageMagick, Lua, Python, yt-dlp, ExifTool,
OCRmyPDF, libvips, libwebp, pngquant, gifsicle, unpaper, LibRaw, Git, wget,
MediaInfo, p7zip, rsync, zstd, and common Python packages.

Before publishing release images, inspect the built image package manifests, for
example:

```bash
docker run --rm <image> dpkg-query -W
docker run --rm <image> python3 -m pip list --format=json
```

## Maintenance Rules

- Add new third-party systems to this file or to the referenced image
  documentation when introducing them.
- Prefer maintained packages with clear licenses.
- Do not commit vendor drops, generated dependency caches, private package
  mirrors, or customer-specific binaries.
- If a package has copyleft, network-service, codec, media, or patent-related
  obligations, document the operational impact in the release notes.
