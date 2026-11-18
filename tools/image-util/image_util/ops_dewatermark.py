"""Corner / box crop and LaMa inpaint (no OpenCV)."""

from __future__ import annotations

from typing import Literal

import numpy as np
from PIL import Image, ImageDraw, ImageFilter

from .ops_lama import inpaint_lama

Corner = Literal["tl", "tr", "bl", "br", "auto"]
Mode = Literal["crop", "inpaint"]
# Kept for CLI/GUI compatibility; LaMa ignores classical methods.
Method = Literal["lama", "telea", "ns"]

Box = tuple[int, int, int, int]  # x, y, w, h in original pixels


def parse_margin(margin: str | int | float, width: int, height: int) -> tuple[int, int]:
    """Return (mw, mh) pixel margins from '8%' or absolute px."""
    if isinstance(margin, (int, float)):
        v = int(margin)
        return max(1, v), max(1, v)
    s = str(margin).strip().lower()
    if s.endswith("%"):
        pct = float(s[:-1]) / 100.0
        return max(1, int(round(width * pct))), max(1, int(round(height * pct)))
    v = int(float(s))
    return max(1, v), max(1, v)


def clamp_box(box: Box, width: int, height: int) -> Box:
    x, y, w, h = box
    x = max(0, min(width - 1, int(x)))
    y = max(0, min(height - 1, int(y)))
    w = max(1, min(width - x, int(w)))
    h = max(1, min(height - y, int(h)))
    return x, y, w, h


def corner_box(
    width: int,
    height: int,
    corner: Corner,
    margin: str | int | float,
    img: Image.Image | None = None,
) -> Box:
    mw, mh = parse_margin(margin, width, height)
    mw = min(mw, width)
    mh = min(mh, height)
    c = corner
    if c == "auto":
        c = _detect_corner(img) if img is not None else "br"
    if c == "tl":
        return 0, 0, mw, mh
    if c == "tr":
        return width - mw, 0, mw, mh
    if c == "bl":
        return 0, height - mh, mw, mh
    return width - mw, height - mh, mw, mh  # br


def _detect_corner(img: Image.Image | None) -> Corner:
    """Pick corner with highest edge energy (numpy / PIL only)."""
    if img is None:
        return "br"
    gray = np.asarray(img.convert("L"), dtype=np.float32)
    h, w = gray.shape
    mw, mh = max(8, w // 12), max(8, h // 12)
    # Simple gradient magnitude.
    gx = np.abs(gray[:, 1:] - gray[:, :-1])
    gy = np.abs(gray[1:, :] - gray[:-1, :])
    edges = np.zeros_like(gray)
    edges[:, 1:] += gx
    edges[1:, :] += gy
    patches = {
        "tl": edges[0:mh, 0:mw],
        "tr": edges[0:mh, w - mw : w],
        "bl": edges[h - mh : h, 0:mw],
        "br": edges[h - mh : h, w - mw : w],
    }
    scores = {k: float(p.mean()) for k, p in patches.items()}
    return max(scores, key=scores.get)  # type: ignore[arg-type]


def resolve_box(
    img: Image.Image,
    *,
    box: Box | None = None,
    corner: Corner = "br",
    margin: str | int | float = "8%",
) -> Box:
    w, h = img.size
    if box is not None:
        return clamp_box(box, w, h)
    return clamp_box(corner_box(w, h, corner, margin, img), w, h)


def crop_box(img: Image.Image, box: Box) -> Image.Image:
    x, y, w, h = clamp_box(box, *img.size)
    return img.crop((x, y, x + w, y + h))


def _roi_bounds(
    x: int, y: int, w: int, h: int, img_w: int, img_h: int, pad: int
) -> tuple[int, int, int, int, int, int]:
    x0 = max(0, x - pad)
    y0 = max(0, y - pad)
    x1 = min(img_w, x + w + pad)
    y1 = min(img_h, y + h + pad)
    return x0, y0, x1, y1, x - x0, y - y0


def _build_mask(roi_w: int, roi_h: int, lx: int, ly: int, bw: int, bh: int, *, dilate: int) -> Image.Image:
    mask = Image.new("L", (roi_w, roi_h), 0)
    draw = ImageDraw.Draw(mask)
    draw.rectangle([lx, ly, lx + bw - 1, ly + bh - 1], fill=255)
    if dilate > 0:
        # Expand hole slightly for cleaner LaMa seams.
        for _ in range(max(1, dilate // 2)):
            mask = mask.filter(ImageFilter.MaxFilter(3))
    return mask


def inpaint_box(
    img: Image.Image,
    box: Box,
    *,
    method: Method = "lama",
    radius: int | None = None,
    refine: bool = True,
) -> Image.Image:
    """Inpaint box with local LaMa on a padded ROI, then paste back."""
    _ = method  # classical telea/ns removed; always LaMa
    img_w, img_h = img.size
    x, y, w, h = clamp_box(box, img_w, img_h)

    # Context pad: more context helps LaMa; clamp for speed on huge images.
    pad = max(32, min(160, max(w, h) * 2))
    if not refine:
        pad = max(24, min(96, max(w, h)))
    if radius is not None:
        pad = max(pad, int(radius) * 4)

    x0, y0, x1, y1, lx, ly = _roi_bounds(x, y, w, h, img_w, img_h, pad)
    roi = img.crop((x0, y0, x1, y1))
    dilate = 3 if refine else 1
    mask = _build_mask(roi.size[0], roi.size[1], lx, ly, w, h, dilate=dilate)

    patched = inpaint_lama(roi, mask)
    out = img.copy()
    # Paste RGB region; keep alpha if present.
    if out.mode == "RGBA" and patched.mode == "RGBA":
        out.paste(patched, (x0, y0))
    else:
        out.paste(patched.convert(out.mode if out.mode in ("RGB", "RGBA") else "RGB"), (x0, y0))
    return out


def dewatermark(
    img: Image.Image,
    *,
    mode: Mode = "inpaint",
    box: Box | None = None,
    corner: Corner = "br",
    margin: str | int | float = "8%",
    method: Method = "lama",
    radius: int | None = None,
    refine: bool = True,
) -> Image.Image:
    rect = resolve_box(img, box=box, corner=corner, margin=margin)
    if mode == "crop":
        return crop_box(img, rect)
    return inpaint_box(img, rect, method=method, radius=radius, refine=refine)
