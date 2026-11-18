"""Lazy local BiRefNet matting (torch / transformers)."""

from __future__ import annotations

import threading
from typing import Any

import numpy as np
from PIL import Image

_lock = threading.Lock()
_model: Any | None = None
_device: Any | None = None
_transform: Any | None = None

_CANDIDATES = ("ZhengPeng7/BiRefNet",)


def birefnet_available() -> bool:
    try:
        import torch  # noqa: F401
        import transformers  # noqa: F401
        import torchvision  # noqa: F401

        return True
    except Exception:
        return False


def get_birefnet(*, lite: bool = False):
    """Load BiRefNet once. lite prefers smaller / faster checkpoint when available."""
    global _model, _device, _transform
    if _model is not None:
        return _model, _device, _transform
    with _lock:
        if _model is not None:
            return _model, _device, _transform
        try:
            import torch
            from torchvision import transforms
            from transformers import AutoModelForImageSegmentation

            _device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
            last_err: Exception | None = None
            model = None
            # Prefer full BiRefNet; lite flag currently still loads same family (weights once).
            for repo in _CANDIDATES:
                try:
                    model = AutoModelForImageSegmentation.from_pretrained(
                        repo, trust_remote_code=True
                    )
                    break
                except Exception as e:  # noqa: BLE001
                    last_err = e
                    continue
            if model is None:
                raise RuntimeError(f"无法加载 BiRefNet: {last_err}")

            # CPU must be float32; HF weights may load as half.
            if _device.type == "cpu":
                model.to(_device, dtype=torch.float32)
            else:
                model.to(_device)
            model.eval()
            _model = model
            _transform = transforms.Compose(
                [
                    transforms.Resize((1024, 1024)),
                    transforms.ToTensor(),
                    transforms.Normalize([0.485, 0.456, 0.406], [0.229, 0.224, 0.225]),
                ]
            )
            _ = lite  # reserved for future lite weights
            return _model, _device, _transform
        except Exception as e:  # noqa: BLE001
            raise RuntimeError(
                "BiRefNet 未能加载。请在 conda game 中安装:\n"
                "  pip install torchvision transformers timm kornia\n"
                f"原始错误: {e}"
            ) from e


def predict_alpha(image: Image.Image, *, lite: bool = False) -> Image.Image:
    """Return L-mode alpha mask (0=bg, 255=fg) same size as image."""
    import torch

    model, device, transform = get_birefnet(lite=lite)
    rgb = image.convert("RGB")
    ow, oh = rgb.size
    dtype = next(model.parameters()).dtype
    inp = transform(rgb).unsqueeze(0).to(device=device, dtype=dtype)
    with torch.no_grad():
        preds = model(inp)[-1].sigmoid().float().cpu()
    pred = preds[0].squeeze()
    # to numpy uint8
    arr = (pred.numpy() * 255.0).clip(0, 255).astype(np.uint8)
    mask = Image.fromarray(arr, mode="L").resize((ow, oh), Image.Resampling.BILINEAR)
    return mask


def matting(image: Image.Image, *, lite: bool = False) -> Image.Image:
    """Return RGBA with BiRefNet alpha."""
    alpha = predict_alpha(image, lite=lite)
    out = image.convert("RGBA")
    out.putalpha(alpha)
    return out
