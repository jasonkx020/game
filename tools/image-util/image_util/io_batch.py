"""Load/save helpers and batch path collection."""

from __future__ import annotations

from pathlib import Path

from PIL import Image, ImageOps

from .ops_convert import save_image

IMAGE_EXTS = {".png", ".jpg", ".jpeg", ".webp", ".bmp", ".tif", ".tiff"}


def load_image(path: str | Path) -> Image.Image:
    path = Path(path)
    img = Image.open(path)
    img = ImageOps.exif_transpose(img)
    # Keep a concrete copy so file handles can close.
    return img.copy()


def collect_inputs(src: str | Path) -> list[Path]:
    src = Path(src)
    if src.is_file():
        return [src]
    if not src.is_dir():
        raise FileNotFoundError(str(src))
    files: list[Path] = []
    for p in sorted(src.rglob("*")):
        if p.is_file() and p.suffix.lower() in IMAGE_EXTS:
            files.append(p)
    return files


def default_out_path(src: Path, out: Path | None, *, suffix: str = "_out", fmt: str | None = None) -> Path:
    if out is None:
        stem = f"{src.stem}{suffix}"
        ext = f".{fmt}" if fmt else src.suffix
        return src.with_name(stem + ext)
    if out.suffix and not out.exists() and (out.parent.exists() or not str(out).endswith(("/", "\\"))):
        # Treat as file path when it has a suffix.
        if out.suffix.lower() in IMAGE_EXTS or fmt:
            return out
    # Directory output
    out.mkdir(parents=True, exist_ok=True)
    ext = f".{fmt}" if fmt else src.suffix
    return out / f"{src.stem}{suffix}{ext}"


def map_batch_out(src_root: Path, src_file: Path, out_dir: Path, *, suffix: str = "_out", fmt: str | None = None) -> Path:
    rel = src_file.relative_to(src_root)
    ext = f".{fmt}" if fmt else src_file.suffix
    dest = out_dir / rel.parent / f"{src_file.stem}{suffix}{ext}"
    dest.parent.mkdir(parents=True, exist_ok=True)
    return dest


__all__ = [
    "IMAGE_EXTS",
    "load_image",
    "collect_inputs",
    "default_out_path",
    "map_batch_out",
    "save_image",
]
