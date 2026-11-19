"""ID photo: BiRefNet cutout + solid background + exam/national size specs."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Literal

from PIL import Image

from .ops_cutout import cutout
from .ops_dewatermark import Box, clamp_box

BgName = Literal["blue", "red", "white"]

BG_PRESETS: dict[str, tuple[int, int, int]] = {
    "blue": (67, 142, 219),
    "red": (255, 0, 0),
    "white": (255, 255, 255),
}


@dataclass(frozen=True)
class IdPhotoSpec:
    label: str
    pixels: tuple[int, int] | None  # None = keep original size
    default_bg: BgName
    note: str = ""


# Scene / national-common electronic photo sizes (approx 300dpi where applicable).
ID_PHOTO_SPECS: dict[str, IdPhotoSpec] = {
    "gwy": IdPhotoSpec(
        label="公务员/事业单位报名",
        pixels=(295, 413),
        default_bg="blue",
        note="国考常见≥295×413、蓝/白底、JPG；确认上传前须用考录系统「照片处理工具」",
    ),
    "1inch": IdPhotoSpec(
        label="标准一寸",
        pixels=(295, 413),
        default_bg="blue",
        note="25×35mm 常见电子尺寸",
    ),
    "1inch_large": IdPhotoSpec(
        label="大一寸",
        pixels=(390, 567),
        default_bg="blue",
        note="33×48mm 常见电子尺寸",
    ),
    "2inch": IdPhotoSpec(
        label="二寸",
        pixels=(413, 579),
        default_bg="blue",
        note="35×49mm 常见电子尺寸",
    ),
    "small2": IdPhotoSpec(
        label="小二寸",
        pixels=(413, 531),
        default_bg="blue",
        note="35×45mm 常见电子尺寸",
    ),
    "idcard": IdPhotoSpec(
        label="身份证（常见电子）",
        pixels=(358, 441),
        default_bg="white",
        note="各地采集系统像素可能略有差异",
    ),
    "passport": IdPhotoSpec(
        label="护照/签证常用",
        pixels=(390, 567),
        default_bg="white",
        note="常见大一寸白底；以受理机关最新要求为准",
    ),
    "social": IdPhotoSpec(
        label="社保卡常见",
        pixels=(358, 441),
        default_bg="blue",
        note="常见电子规格，以当地社保要求为准",
    ),
    "driver": IdPhotoSpec(
        label="驾驶证常见",
        pixels=(260, 378),
        default_bg="white",
        note="常见小一寸电子规格，以车管所要求为准",
    ),
    "keep": IdPhotoSpec(
        label="保持原尺寸",
        pixels=None,
        default_bg="blue",
        note="只换底不改像素",
    ),
}

# Backward-compatible pixel map
SIZE_PRESETS: dict[str, tuple[int, int] | None] = {k: v.pixels for k, v in ID_PHOTO_SPECS.items()}

SizeKey = Literal[
    "gwy",
    "1inch",
    "1inch_large",
    "2inch",
    "small2",
    "idcard",
    "passport",
    "social",
    "driver",
    "keep",
]


def list_id_photo_specs() -> list[tuple[str, IdPhotoSpec]]:
    return list(ID_PHOTO_SPECS.items())


def get_spec(size_key: str) -> IdPhotoSpec:
    key = str(size_key).strip().lower()
    if key not in ID_PHOTO_SPECS:
        keys = "|".join(ID_PHOTO_SPECS)
        raise ValueError(f"unknown size: {size_key}; use {keys}")
    return ID_PHOTO_SPECS[key]


def resolve_bg_rgb(
    bg: str | tuple[int, int, int] | None = None,
    *,
    bg_rgb: tuple[int, int, int] | None = None,
) -> tuple[int, int, int]:
    if bg_rgb is not None:
        r, g, b = bg_rgb
        return max(0, min(255, int(r))), max(0, min(255, int(g))), max(0, min(255, int(b)))
    name = (bg or "blue").strip().lower()
    if name not in BG_PRESETS:
        raise ValueError(f"unknown bg preset: {bg}; use blue|red|white or bg_rgb")
    return BG_PRESETS[name]


def expand_box_to_aspect(
    box: Box,
    img_w: int,
    img_h: int,
    tw: int,
    th: int,
) -> Box:
    """
    Expand (or fit) box to target aspect tw:th, centered on the selection.
    Prefers growing so the original box stays inside; shrinks uniformly if image bounds require it.
    """
    x, y, w, h = box
    img_w, img_h = max(1, int(img_w)), max(1, int(img_h))
    tw, th = max(1, int(tw)), max(1, int(th))
    aspect = tw / float(th)

    fw = max(float(w), float(h) * aspect)
    fh = fw / aspect
    if fh < float(h):
        fh = float(h)
        fw = fh * aspect

    if fw > img_w or fh > img_h:
        scale = min(img_w / fw, img_h / fh)
        fw *= scale
        fh *= scale

    fw = max(1.0, min(fw, float(img_w)))
    fh = max(1.0, min(fh, float(img_h)))
    if fw / fh > aspect:
        fw = fh * aspect
    else:
        fh = fw / aspect
    fw = max(1.0, min(fw, float(img_w)))
    fh = max(1.0, min(fh, float(img_h)))

    cx = x + w / 2.0
    cy = y + h / 2.0
    nx = cx - fw / 2.0
    ny = cy - fh / 2.0
    nx = max(0.0, min(nx, img_w - fw))
    ny = max(0.0, min(ny, img_h - fh))
    return clamp_box((int(round(nx)), int(round(ny)), int(round(fw)), int(round(fh))), img_w, img_h)


def max_inscribed_aspect_box(img_w: int, img_h: int, tw: int, th: int) -> Box:
    """Largest axis-aligned box with aspect tw:th centered in the image."""
    img_w, img_h = max(1, int(img_w)), max(1, int(img_h))
    tw, th = max(1, int(tw)), max(1, int(th))
    aspect = tw / float(th)
    if img_w / float(img_h) >= aspect:
        fh = float(img_h)
        fw = fh * aspect
    else:
        fw = float(img_w)
        fh = fw / aspect
    nx = (img_w - fw) / 2.0
    ny = (img_h - fh) / 2.0
    return clamp_box((int(round(nx)), int(round(ny)), int(round(fw)), int(round(fh))), img_w, img_h)


def composite_on_bg(rgba: Image.Image, bg_rgb: tuple[int, int, int]) -> Image.Image:
    base = Image.new("RGBA", rgba.size, (*bg_rgb, 255))
    layered = Image.alpha_composite(base, rgba.convert("RGBA"))
    return layered.convert("RGB")


def make_id_photo(
    img: Image.Image,
    *,
    bg_rgb: tuple[int, int, int] = BG_PRESETS["blue"],
    size_key: SizeKey | str = "gwy",
    box: Box | None = None,
) -> Image.Image:
    """Cut subject, solid bg, frame by aspect-adjusted box, resize to ID pixels."""
    spec = get_spec(size_key)
    target = spec.pixels
    iw, ih = img.size

    if target is None:
        rgba = cutout(img, mode="birefnet", box=box, lite=False)
        return composite_on_bg(rgba, bg_rgb)

    tw, th = target
    if box is not None:
        adj = expand_box_to_aspect(box, iw, ih, tw, th)
    else:
        adj = max_inscribed_aspect_box(iw, ih, tw, th)

    rgba = cutout(img, mode="birefnet", box=adj, lite=False)
    out = composite_on_bg(rgba, bg_rgb)
    ax, ay, aw, ah = adj
    cropped = out.crop((ax, ay, ax + aw, ay + ah))
    return cropped.resize((tw, th), Image.Resampling.LANCZOS)
