"""Preview scaling and selection overlay helpers."""

from __future__ import annotations

from dataclasses import dataclass

from PIL import Image, ImageDraw

from .ops_dewatermark import Box, clamp_box


@dataclass
class DisplayTransform:
    """Maps between canvas/display coords and original image pixels."""

    scale: float
    offset_x: int
    offset_y: int
    disp_w: int
    disp_h: int
    orig_w: int
    orig_h: int

    def canvas_to_image(self, cx: float, cy: float) -> tuple[int, int]:
        ix = int(round((cx - self.offset_x) / self.scale))
        iy = int(round((cy - self.offset_y) / self.scale))
        ix = max(0, min(self.orig_w - 1, ix))
        iy = max(0, min(self.orig_h - 1, iy))
        return ix, iy

    def image_to_canvas(self, ix: float, iy: float) -> tuple[float, float]:
        cx = self.offset_x + ix * self.scale
        cy = self.offset_y + iy * self.scale
        return cx, cy

    def canvas_box_to_image(self, x0: float, y0: float, x1: float, y1: float) -> Box:
        ax, ay = self.canvas_to_image(min(x0, x1), min(y0, y1))
        bx, by = self.canvas_to_image(max(x0, x1), max(y0, y1))
        w = max(1, bx - ax + 1)
        h = max(1, by - ay + 1)
        return clamp_box((ax, ay, w, h), self.orig_w, self.orig_h)


def fit_preview(
    img: Image.Image,
    max_side: int = 4096,
    canvas_w: int | None = None,
    canvas_h: int | None = None,
    *,
    zoom: float = 1.0,
    pan_x: float = 0.0,
    pan_y: float = 0.0,
) -> tuple[Image.Image, DisplayTransform]:
    """Fit image into canvas, then apply zoom (1=fit) and pan offsets in canvas pixels."""
    ow, oh = img.size
    zoom = max(0.25, min(16.0, float(zoom)))

    if canvas_w is not None and canvas_h is not None and canvas_w > 0 and canvas_h > 0:
        base = min(canvas_w / float(ow), canvas_h / float(oh))
        scale = base * zoom
    else:
        scale = 1.0 * zoom

    dw = max(1, int(round(ow * scale)))
    dh = max(1, int(round(oh * scale)))
    # Soft cap to keep Tk PhotoImage responsive; keep scale in sync.
    if max_side > 0 and max(dw, dh) > max_side:
        cap = max_side / float(max(dw, dh))
        scale *= cap
        dw = max(1, int(round(ow * scale)))
        dh = max(1, int(round(oh * scale)))

    if canvas_w is not None and canvas_h is not None and canvas_w > 0 and canvas_h > 0:
        ox = int(round((canvas_w - dw) / 2.0 + pan_x))
        oy = int(round((canvas_h - dh) / 2.0 + pan_y))
    else:
        ox = int(round(pan_x))
        oy = int(round(pan_y))

    if (dw, dh) != (ow, oh):
        preview = img.resize((dw, dh), Image.Resampling.BILINEAR if zoom > 1.5 else Image.Resampling.LANCZOS)
    else:
        preview = img.copy()

    tf = DisplayTransform(
        scale=scale,
        offset_x=ox,
        offset_y=oy,
        disp_w=dw,
        disp_h=dh,
        orig_w=ow,
        orig_h=oh,
    )
    return preview, tf


def overlay_box(
    img: Image.Image,
    box: Box,
    *,
    color: tuple[int, int, int, int] = (255, 64, 64, 90),
    outline: tuple[int, int, int, int] = (255, 40, 40, 220),
) -> Image.Image:
    """Draw semi-transparent rect on a copy (image-pixel coordinates)."""
    base = img.convert("RGBA")
    overlay = Image.new("RGBA", base.size, (0, 0, 0, 0))
    draw = ImageDraw.Draw(overlay)
    x, y, w, h = clamp_box(box, *base.size)
    draw.rectangle([x, y, x + w - 1, y + h - 1], fill=color, outline=outline, width=2)
    return Image.alpha_composite(base, overlay)


def checkerboard(size: tuple[int, int], cell: int = 12) -> Image.Image:
    """Opaque checkerboard for previewing transparency."""
    w, h = size
    img = Image.new("RGB", (w, h), (200, 200, 200))
    draw = ImageDraw.Draw(img)
    c2 = (240, 240, 240)
    for y in range(0, h, cell):
        for x in range(0, w, cell):
            if ((x // cell) + (y // cell)) % 2 == 0:
                draw.rectangle([x, y, min(x + cell - 1, w - 1), min(y + cell - 1, h - 1)], fill=c2)
    return img


def composite_on_checker(img: Image.Image, cell: int = 12) -> Image.Image:
    """Composite RGBA onto checkerboard for display."""
    rgba = img.convert("RGBA")
    bg = checkerboard(rgba.size, cell=cell)
    return Image.alpha_composite(bg.convert("RGBA"), rgba).convert("RGB")


def clamp_pan(
    pan_x: float,
    pan_y: float,
    *,
    orig_w: int,
    orig_h: int,
    canvas_w: int,
    canvas_h: int,
    zoom: float,
) -> tuple[float, float]:
    """Keep pan in range so every corner of the image can be brought into view."""
    if orig_w <= 0 or orig_h <= 0 or canvas_w <= 0 or canvas_h <= 0:
        return 0.0, 0.0
    zoom = max(0.25, min(16.0, float(zoom)))
    base = min(canvas_w / float(orig_w), canvas_h / float(orig_h))
    scale = base * zoom
    dw = orig_w * scale
    dh = orig_h * scale

    def _axis(pan: float, canvas: float, disp: float) -> float:
        if disp <= canvas:
            # Image fits: keep near centered (small slack).
            slack = max(0.0, (canvas - disp) / 2.0)
            return max(-slack, min(slack, pan))
        # Image larger than canvas: allow panning until each edge is reachable.
        lim = (disp - canvas) / 2.0
        return max(-lim, min(lim, pan))

    return _axis(pan_x, canvas_w, dw), _axis(pan_y, canvas_h, dh)
