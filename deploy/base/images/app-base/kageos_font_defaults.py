"""Best-effort Python font defaults for the app-base image.

This module is imported from sitecustomize at Python startup. Keep it silent:
generated user code should not fail just because font defaults cannot be
installed in an unusual environment.
"""

from __future__ import annotations

import os
from pathlib import Path


def _truthy(value: str | None, default: bool = True) -> bool:
    if value is None:
        return default
    return value.strip().lower() not in {"0", "false", "no", "off"}


def _register_reportlab_cjk_fonts() -> None:
    if not _truthy(os.environ.get("KAGEOS_REPORTLAB_CJK_AUTOREGISTER")):
        return

    font_path = Path(
        os.environ.get(
            "KAGEOS_REPORTLAB_CJK_FONT",
            "/usr/local/share/fonts/kageos/wqy-zenhei.ttc",
        )
    )
    if not font_path.is_file():
        return

    try:
        from reportlab.pdfbase import pdfmetrics
        from reportlab.pdfbase.ttfonts import TTFont
    except Exception:
        return

    # ReportLab's built-in PDF fonts are WinAnsi-only, so Chinese drawn with
    # Helvetica-Bold becomes black squares. Registering these common names lets
    # generated scripts that use standard font names still render CJK text.
    names = (
        "KageOS-CJK",
        "Helvetica",
        "Helvetica-Bold",
        "Helvetica-Oblique",
        "Helvetica-BoldOblique",
        "Times-Roman",
        "Times-Bold",
        "Times-Italic",
        "Times-BoldItalic",
        "Courier",
        "Courier-Bold",
        "Courier-Oblique",
        "Courier-BoldOblique",
    )
    for name in names:
        try:
            pdfmetrics.registerFont(TTFont(name, str(font_path)))
        except Exception:
            pass


def _patch_reportlab_ttfont_fallback() -> None:
    if not _truthy(os.environ.get("KAGEOS_REPORTLAB_CJK_FALLBACK")):
        return

    fallback_path = Path(
        os.environ.get(
            "KAGEOS_REPORTLAB_CJK_FONT",
            "/usr/local/share/fonts/kageos/wqy-zenhei.ttc",
        )
    )
    if not fallback_path.is_file():
        return

    try:
        from reportlab.pdfbase import ttfonts
    except Exception:
        return

    original_ttfont = ttfonts.TTFont

    class KageOSTTFont(original_ttfont):
        def __init__(self, name, filename, *args, **kwargs):
            try:
                super().__init__(name, filename, *args, **kwargs)
            except Exception as exc:
                if "postscript outlines are not supported" not in str(exc).lower():
                    raise
                super().__init__(name, str(fallback_path), *args, **kwargs)

    ttfonts.TTFont = KageOSTTFont


def _patch_pillow_default_font() -> None:
    if not _truthy(os.environ.get("KAGEOS_PIL_CJK_DEFAULT")):
        return

    font_path = Path(
        os.environ.get(
            "KAGEOS_CJK_FONT",
            "/usr/local/share/fonts/kageos/NotoSansCJKSC-Regular.otf",
        )
    )
    if not font_path.is_file():
        return

    try:
        from PIL import ImageFont
    except Exception:
        return

    original_load_default = ImageFont.load_default
    font_index = int(os.environ.get("KAGEOS_CJK_FONT_INDEX", "0"))

    def load_default(size=None):
        try:
            return ImageFont.truetype(
                str(font_path),
                size or 10,
                index=font_index,
            )
        except Exception:
            return original_load_default(size=size)

    ImageFont.load_default = load_default


_patch_reportlab_ttfont_fallback()
_register_reportlab_cjk_fonts()
_patch_pillow_default_font()
