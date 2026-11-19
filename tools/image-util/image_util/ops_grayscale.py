"""Grayscale / desaturation helpers."""

from __future__ import annotations

from PIL import Image, ImageEnhance

from .ops_dewatermark import Box, clamp_box


def _desaturate(img: Image.Image, amount: float) -> Image.Image:
    """Lower color saturation. amount 0=unchanged, 1=full grayscale."""
    amount = max(0.0, min(1.0, float(amount)))
    if amount <= 0:
        return img.copy()
    # ImageEnhance.Color: 0=gray, 1=original
    factor = 1.0 - amount
    if img.mode == "RGBA":
        rgb = img.convert("RGB")
        gray = ImageEnhance.Color(rgb).enhance(factor)
        out = gray.convert("RGBA")
        out.putalpha(img.getchannel("A"))
        return out
    if img.mode == "LA":
        l = img.getchannel("L")
        a = img.getchannel("A")
        if amount >= 1.0:
            return Image.merge("LA", (l, a))
        rgb = Image.merge("RGB", (l, l, l))
        # already gray — color enhance is no-op; return copy
        out = Image.merge("LA", (l, a))
        return out
    if img.mode != "RGB":
        base = img.convert("RGB")
        gray = ImageEnhance.Color(base).enhance(factor)
        return gray.convert(img.mode) if img.mode in ("L", "P") else gray
    return ImageEnhance.Color(img).enhance(factor)


def grayscale_image(
    img: Image.Image,
    *,
    amount: float = 1.0,
    box: Box | None = None,
) -> Image.Image:
    """Desaturate whole image or a clamped box region. Preserves mode/alpha."""
    amount = max(0.0, min(1.0, float(amount)))
    if amount <= 0:
        return img.copy()

    if box is None:
        return _desaturate(img, amount)

    x, y, w, h = clamp_box(box, *img.size)
    region = img.crop((x, y, x + w, y + h))
    desat = _desaturate(region, amount)
    out = img.copy()
    out.paste(desat, (x, y))
    return out
