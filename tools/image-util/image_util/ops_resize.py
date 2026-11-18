"""Image resize helpers."""

from __future__ import annotations

from PIL import Image


def resize_image(
    img: Image.Image,
    *,
    width: int | None = None,
    height: int | None = None,
    max_side: int | None = None,
    keep_aspect: bool = True,
) -> Image.Image:
    """Resize by explicit size and/or max longest side."""
    w, h = img.size
    if w <= 0 or h <= 0:
        return img.copy()

    target_w, target_h = w, h

    if max_side is not None and max_side > 0:
        longest = max(w, h)
        if longest > max_side:
            scale = max_side / float(longest)
            target_w = max(1, int(round(w * scale)))
            target_h = max(1, int(round(h * scale)))
            w, h = target_w, target_h

    if width is not None or height is not None:
        if keep_aspect:
            if width is not None and height is not None:
                scale = min(width / float(w), height / float(h))
                target_w = max(1, int(round(w * scale)))
                target_h = max(1, int(round(h * scale)))
            elif width is not None:
                scale = width / float(w)
                target_w = max(1, int(width))
                target_h = max(1, int(round(h * scale)))
            else:
                scale = height / float(h)  # type: ignore[arg-type]
                target_h = max(1, int(height))  # type: ignore[arg-type]
                target_w = max(1, int(round(w * scale)))
        else:
            target_w = max(1, int(width if width is not None else w))
            target_h = max(1, int(height if height is not None else h))

    if (target_w, target_h) == img.size:
        return img.copy()
    return img.resize((target_w, target_h), Image.Resampling.LANCZOS)
