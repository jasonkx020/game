"""Lazy local LaMa inpainting (no OpenCV)."""

from __future__ import annotations

import threading
from typing import Any

from PIL import Image

_lock = threading.Lock()
_lama: Any | None = None
_lama_error: str | None = None


def lama_available() -> bool:
    try:
        import torch  # noqa: F401
        from simple_lama_inpainting import SimpleLama  # noqa: F401

        return True
    except Exception:
        return False


def get_lama():
    """Load SimpleLama once (downloads weights on first use)."""
    global _lama, _lama_error
    if _lama is not None:
        return _lama
    with _lock:
        if _lama is not None:
            return _lama
        try:
            import torch
            from simple_lama_inpainting import SimpleLama

            device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
            _lama = SimpleLama(device=device)
            _lama_error = None
            return _lama
        except Exception as e:  # noqa: BLE001
            _lama_error = str(e)
            raise RuntimeError(
                "LaMa 未能加载。请在 conda game 中安装: "
                "pip install torch simple-lama-inpainting\n"
                f"原始错误: {_lama_error}"
            ) from e


def inpaint_lama(image: Image.Image, mask: Image.Image) -> Image.Image:
    """
    Run LaMa on RGB image + L mask (white = hole).
    Returns RGB/RGBA matching input mode as much as possible.
    """
    lama = get_lama()
    rgb = image.convert("RGB")
    m = mask.convert("L")
    # LaMa prefers sizes divisible by 8.
    w, h = rgb.size
    nw = max(8, (w + 7) // 8 * 8)
    nh = max(8, (h + 7) // 8 * 8)
    if (nw, nh) != (w, h):
        rgb_p = Image.new("RGB", (nw, nh), (0, 0, 0))
        rgb_p.paste(rgb, (0, 0))
        m_p = Image.new("L", (nw, nh), 0)
        m_p.paste(m, (0, 0))
        out = lama(rgb_p, m_p)
        if not isinstance(out, Image.Image):
            out = Image.fromarray(out)
        out = out.convert("RGB").crop((0, 0, w, h))
    else:
        out = lama(rgb, m)
        if not isinstance(out, Image.Image):
            out = Image.fromarray(out)
        out = out.convert("RGB")

    if image.mode == "RGBA":
        alpha = image.getchannel("A")
        out = out.convert("RGBA")
        out.putalpha(alpha)
    return out
