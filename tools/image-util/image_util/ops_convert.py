"""Format conversion helpers."""

from __future__ import annotations

from pathlib import Path

from PIL import Image

FORMAT_MAP = {
    "png": "PNG",
    "jpg": "JPEG",
    "jpeg": "JPEG",
    "webp": "WEBP",
    "bmp": "BMP",
}


def normalize_format(fmt: str) -> str:
    key = fmt.strip().lower().lstrip(".")
    if key not in FORMAT_MAP:
        raise ValueError(f"unsupported format: {fmt}")
    return key


def prepare_for_format(img: Image.Image, fmt: str) -> Image.Image:
    """Adjust mode for target format (e.g. drop alpha for JPEG)."""
    fmt = normalize_format(fmt)
    pil_fmt = FORMAT_MAP[fmt]
    out = img
    if pil_fmt == "JPEG":
        if out.mode in ("RGBA", "LA") or (out.mode == "P" and "transparency" in out.info):
            rgba = out.convert("RGBA")
            background = Image.new("RGB", rgba.size, (255, 255, 255))
            background.paste(rgba, mask=rgba.split()[-1])
            out = background
        elif out.mode != "RGB":
            out = out.convert("RGB")
    elif pil_fmt == "BMP":
        if out.mode in ("RGBA", "LA", "P"):
            out = out.convert("RGB")
    elif out.mode == "P":
        out = out.convert("RGBA")
    return out


def save_image(
    img: Image.Image,
    path: str | Path,
    *,
    fmt: str | None = None,
    quality: int = 90,
) -> Path:
    path = Path(path)
    if fmt is None:
        fmt = path.suffix.lstrip(".") or "png"
    fmt = normalize_format(fmt)
    pil_fmt = FORMAT_MAP[fmt]
    if path.suffix.lower().lstrip(".") != fmt and fmt == "jpg":
        path = path.with_suffix(".jpg")
    elif path.suffix.lower().lstrip(".") not in (fmt, "jpeg" if fmt == "jpg" else fmt):
        path = path.with_suffix(f".{fmt}")

    prepared = prepare_for_format(img, fmt)
    path.parent.mkdir(parents=True, exist_ok=True)
    kwargs: dict = {}
    if pil_fmt in ("JPEG", "WEBP"):
        kwargs["quality"] = max(1, min(100, int(quality)))
    if pil_fmt == "JPEG":
        kwargs["optimize"] = True
    prepared.save(path, format=pil_fmt, **kwargs)
    return path
