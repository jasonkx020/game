"""Image crop helpers."""

from __future__ import annotations

from PIL import Image

from .ops_dewatermark import Box, clamp_box


def crop_image(img: Image.Image, box: Box) -> Image.Image:
    x, y, w, h = clamp_box(box, *img.size)
    return img.crop((x, y, x + w, y + h))


def parse_box_xywh(s: str) -> Box:
    parts = [p.strip() for p in s.replace("x", ",").split(",")]
    if len(parts) != 4:
        raise ValueError("box must be x,y,w,h")
    x, y, w, h = (int(p) for p in parts)
    return x, y, w, h
