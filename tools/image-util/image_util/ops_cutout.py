"""Local cutout / transparent background.

Primary: BiRefNet (torch). Optional classical: flood / colorkey (numpy only).
"""

from __future__ import annotations

from typing import Literal

import numpy as np
from PIL import Image

from .ops_birefnet import matting as birefnet_matting
from .ops_dewatermark import Box, clamp_box

CutoutMode = Literal["birefnet", "flood", "colorkey"]


def _apply_alpha(img: Image.Image, alpha: np.ndarray) -> Image.Image:
    rgba = img.convert("RGBA")
    arr = np.asarray(rgba).copy()
    a = np.clip(alpha, 0, 255).astype(np.uint8)
    if a.ndim != 2 or a.shape[:2] != arr.shape[:2]:
        raise ValueError("alpha mask size mismatch")
    arr[:, :, 3] = a
    return Image.fromarray(arr, mode="RGBA")


def cutout_birefnet(
    img: Image.Image,
    box: Box | None = None,
    *,
    lite: bool = False,
) -> Image.Image:
    """
    BiRefNet matting.
    If box is set, run on padded ROI then paste alpha back (faster, subject-focused).
    """
    img_w, img_h = img.size
    if box is None:
        return birefnet_matting(img, lite=lite)

    x, y, w, h = clamp_box(box, img_w, img_h)
    pad = max(24, min(128, max(w, h) // 2))
    x0 = max(0, x - pad)
    y0 = max(0, y - pad)
    x1 = min(img_w, x + w + pad)
    y1 = min(img_h, y + h + pad)
    roi = img.crop((x0, y0, x1, y1))
    roi_rgba = birefnet_matting(roi, lite=lite)

    # Full transparent canvas; paste ROI alpha result.
    out = Image.new("RGBA", (img_w, img_h), (0, 0, 0, 0))
    out.paste(roi_rgba, (x0, y0))
    # Outside ROI stays transparent — keeps subject island; if user boxed subject this is desired.
    # Soften: also keep original RGB outside with 0 alpha already.
    return out


def cutout_flood(
    img: Image.Image,
    *,
    tolerance: int = 30,
) -> Image.Image:
    """Flood from borders using numpy BFS (no OpenCV)."""
    rgb = np.asarray(img.convert("RGB"), dtype=np.int16)
    h, w = rgb.shape[:2]
    tol = max(0, min(255, int(tolerance)))
    alpha = np.full((h, w), 255, dtype=np.uint8)
    visited = np.zeros((h, w), dtype=np.uint8)

    seeds = [
        (0, 0),
        (w - 1, 0),
        (0, h - 1),
        (w - 1, h - 1),
        (w // 2, 0),
        (w // 2, h - 1),
        (0, h // 2),
        (w - 1, h // 2),
    ]
    for sx, sy in seeds:
        if visited[sy, sx]:
            continue
        base = rgb[sy, sx]
        stack = [(sx, sy)]
        visited[sy, sx] = 1
        while stack:
            x, y = stack.pop()
            if int(np.max(np.abs(rgb[y, x] - base))) > tol:
                continue
            alpha[y, x] = 0
            for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
                if 0 <= nx < w and 0 <= ny < h and not visited[ny, nx]:
                    visited[ny, nx] = 1
                    stack.append((nx, ny))
    return _apply_alpha(img, alpha)


def cutout_colorkey(
    img: Image.Image,
    key_rgb: tuple[int, int, int] = (255, 255, 255),
    *,
    tolerance: int = 30,
) -> Image.Image:
    """Make pixels close to key_rgb transparent."""
    rgb = np.asarray(img.convert("RGB"), dtype=np.int16)
    key = np.array(key_rgb, dtype=np.int16).reshape(1, 1, 3)
    dist = np.max(np.abs(rgb - key), axis=2)
    alpha = np.where(dist <= max(0, int(tolerance)), 0, 255).astype(np.uint8)
    return _apply_alpha(img, alpha)


def cutout(
    img: Image.Image,
    *,
    mode: CutoutMode = "birefnet",
    box: Box | None = None,
    tolerance: int = 30,
    key_rgb: tuple[int, int, int] = (255, 255, 255),
    iters: int = 5,
    lite: bool = False,
) -> Image.Image:
    _ = iters  # legacy grabcut param ignored
    if mode in ("birefnet", "grabcut"):  # grabcut alias → BiRefNet
        return cutout_birefnet(img, box, lite=lite)
    if mode == "flood":
        return cutout_flood(img, tolerance=tolerance)
    if mode == "colorkey":
        return cutout_colorkey(img, key_rgb, tolerance=tolerance)
    raise ValueError(f"unsupported cutout mode: {mode}")


def parse_rgb(s: str) -> tuple[int, int, int]:
    parts = [p.strip() for p in s.replace(" ", "").split(",")]
    if len(parts) != 3:
        raise ValueError("RGB must be R,G,B")
    r, g, b = (int(p) for p in parts)
    return max(0, min(255, r)), max(0, min(255, g)), max(0, min(255, b))
