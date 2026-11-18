"""CLI for batch resize / convert / dewatermark."""

from __future__ import annotations

import argparse
from pathlib import Path

from .io_batch import collect_inputs, default_out_path, load_image, map_batch_out, save_image
from .ops_convert import normalize_format
from .ops_crop import crop_image
from .ops_cutout import cutout, parse_rgb
from .ops_dewatermark import Box, dewatermark
from .ops_resize import resize_image


def _parse_box(s: str) -> Box:
    parts = [p.strip() for p in s.replace("x", ",").split(",")]
    if len(parts) != 4:
        raise argparse.ArgumentTypeError("box must be x,y,w,h")
    x, y, w, h = (int(p) for p in parts)
    return x, y, w, h


def _parse_rgb_arg(s: str) -> tuple[int, int, int]:
    try:
        return parse_rgb(s)
    except ValueError as e:
        raise argparse.ArgumentTypeError(str(e)) from e


def _add_io(p: argparse.ArgumentParser) -> None:
    p.add_argument("input", help="image file or directory")
    p.add_argument("-o", "--output", default=None, help="output file or directory")
    p.add_argument("--inplace", action="store_true", help="overwrite inputs (dangerous)")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="image_util",
        description="Local image util (GUI by default). Use subcommands for batch CLI.",
    )
    sub = parser.add_subparsers(dest="cmd")

    g = sub.add_parser("gui", help="open visual UI")
    g.set_defaults(_fn="gui")

    r = sub.add_parser("resize", help="resize images")
    _add_io(r)
    r.add_argument("--width", type=int, default=None)
    r.add_argument("--height", type=int, default=None)
    r.add_argument("--max-side", type=int, default=None)
    r.add_argument("--keep-aspect", action=argparse.BooleanOptionalAction, default=True)
    r.add_argument("--format", default=None, help="optional output format")
    r.add_argument("--quality", type=int, default=90)
    r.set_defaults(_fn="resize")

    c = sub.add_parser("convert", help="convert format")
    _add_io(c)
    c.add_argument("--format", required=True)
    c.add_argument("--quality", type=int, default=90)
    c.set_defaults(_fn="convert")

    d = sub.add_parser("dewatermark", help="corner or box crop/inpaint")
    _add_io(d)
    d.add_argument("--mode", choices=("crop", "inpaint"), default="inpaint")
    d.add_argument("--corner", choices=("tl", "tr", "bl", "br", "auto"), default="br")
    d.add_argument("--margin", default="8%")
    d.add_argument("--box", type=_parse_box, default=None, help="x,y,w,h original pixels (wins over corner)")
    d.add_argument("--method", choices=("lama",), default="lama", help="LaMa only (OpenCV removed)")
    d.add_argument("--refine", action=argparse.BooleanOptionalAction, default=True, help="larger LaMa context pad")
    d.add_argument("--format", default=None)
    d.add_argument("--quality", type=int, default=90)
    d.set_defaults(_fn="dewatermark")

    m = sub.add_parser("cutout", help="cutout / transparent background (birefnet|flood|colorkey)")
    _add_io(m)
    m.add_argument("--mode", choices=("birefnet", "flood", "colorkey"), default="birefnet")
    m.add_argument("--box", type=_parse_box, default=None, help="optional subject ROI for BiRefNet: x,y,w,h")
    m.add_argument("--tolerance", type=int, default=30, help="flood/colorkey tolerance")
    m.add_argument("--key", type=_parse_rgb_arg, default="255,255,255", help="colorkey RGB, e.g. 255,255,255")
    m.add_argument("--format", default="png")
    m.add_argument("--quality", type=int, default=90)
    m.set_defaults(_fn="cutout")

    cr = sub.add_parser("crop", help="crop by x,y,w,h")
    _add_io(cr)
    cr.add_argument("--box", type=_parse_box, required=True, help="x,y,w,h original pixels")
    cr.add_argument("--format", default=None)
    cr.add_argument("--quality", type=int, default=90)
    cr.set_defaults(_fn="crop")

    p = sub.add_parser("process", help="optional dewatermark/cutout/crop then resize/convert")
    _add_io(p)
    p.add_argument("--mode", choices=("crop", "inpaint"), default="inpaint")
    p.add_argument("--corner", choices=("tl", "tr", "bl", "br", "auto"), default="br")
    p.add_argument("--margin", default="8%")
    p.add_argument("--box", type=_parse_box, default=None)
    p.add_argument("--method", choices=("lama",), default="lama")
    p.add_argument("--refine", action=argparse.BooleanOptionalAction, default=True)
    p.add_argument("--cutout", choices=("birefnet", "flood", "colorkey"), default=None)
    p.add_argument("--tolerance", type=int, default=30)
    p.add_argument("--key", type=_parse_rgb_arg, default="255,255,255")
    p.add_argument("--crop-box", type=_parse_box, default=None, help="final crop x,y,w,h")
    p.add_argument("--width", type=int, default=None)
    p.add_argument("--height", type=int, default=None)
    p.add_argument("--max-side", type=int, default=None)
    p.add_argument("--keep-aspect", action=argparse.BooleanOptionalAction, default=True)
    p.add_argument("--format", default=None)
    p.add_argument("--quality", type=int, default=90)
    p.add_argument("--no-dewatermark", action="store_true")
    p.set_defaults(_fn="process")

    return parser


def _iter_jobs(args: argparse.Namespace) -> list[tuple[Path, Path]]:
    src = Path(args.input)
    files = collect_inputs(src)
    if not files:
        raise SystemExit(f"no images found: {src}")
    out = Path(args.output) if args.output else None
    jobs: list[tuple[Path, Path]] = []
    fmt = getattr(args, "format", None)
    if fmt:
        fmt = normalize_format(fmt)

    if src.is_dir():
        out_dir = out or (src / "_out")
        out_dir.mkdir(parents=True, exist_ok=True)
        for f in files:
            if args.inplace:
                dest = f if not fmt else f.with_suffix(f".{fmt}")
            else:
                dest = map_batch_out(src, f, out_dir, fmt=fmt)
            jobs.append((f, dest))
        return jobs

    # single file
    f = files[0]
    if args.inplace:
        dest = f if not fmt else f.with_suffix(f".{fmt}")
    else:
        dest = default_out_path(f, out, fmt=fmt)
    jobs.append((f, dest))
    return jobs


def _run_resize(args: argparse.Namespace) -> int:
    for src, dest in _iter_jobs(args):
        img = load_image(src)
        out = resize_image(
            img,
            width=args.width,
            height=args.height,
            max_side=args.max_side,
            keep_aspect=args.keep_aspect,
        )
        save_image(out, dest, fmt=args.format or dest.suffix.lstrip("."), quality=args.quality)
        print(f"OK resize {src} -> {dest}")
    return 0


def _run_convert(args: argparse.Namespace) -> int:
    fmt = normalize_format(args.format)
    for src, dest in _iter_jobs(args):
        img = load_image(src)
        save_image(img, dest, fmt=fmt, quality=args.quality)
        print(f"OK convert {src} -> {dest}")
    return 0


def _run_dewatermark(args: argparse.Namespace) -> int:
    for src, dest in _iter_jobs(args):
        img = load_image(src)
        out = dewatermark(
            img,
            mode=args.mode,
            box=args.box,
            corner=args.corner,
            margin=args.margin,
            method=args.method,
            refine=args.refine,
        )
        save_image(out, dest, fmt=args.format or dest.suffix.lstrip("."), quality=args.quality)
        print(f"OK dewatermark {src} -> {dest}")
    return 0


def _run_cutout(args: argparse.Namespace) -> int:
    key = args.key if isinstance(args.key, tuple) else parse_rgb(str(args.key))
    for src, dest in _iter_jobs(args):
        img = load_image(src)
        out = cutout(
            img,
            mode=args.mode,
            box=args.box,
            tolerance=args.tolerance,
            key_rgb=key,
        )
        fmt = args.format or "png"
        if normalize_format(fmt) not in ("png", "webp"):
            fmt = "png"
            dest = dest.with_suffix(".png")
        save_image(out, dest, fmt=fmt, quality=args.quality)
        print(f"OK cutout {src} -> {dest}")
    return 0


def _run_crop(args: argparse.Namespace) -> int:
    for src, dest in _iter_jobs(args):
        img = load_image(src)
        out = crop_image(img, args.box)
        save_image(out, dest, fmt=args.format or dest.suffix.lstrip("."), quality=args.quality)
        print(f"OK crop {src} -> {dest} size={out.size}")
    return 0


def _run_process(args: argparse.Namespace) -> int:
    key = args.key if isinstance(args.key, tuple) else parse_rgb(str(args.key))
    for src, dest in _iter_jobs(args):
        img = load_image(src)
        out = img
        if not args.no_dewatermark:
            out = dewatermark(
                out,
                mode=args.mode,
                box=args.box,
                corner=args.corner,
                margin=args.margin,
                method=args.method,
                refine=args.refine,
            )
        if args.cutout:
            out = cutout(
                out,
                mode=args.cutout,
                box=args.box,
                tolerance=args.tolerance,
                key_rgb=key,
            )
        if args.crop_box is not None:
            out = crop_image(out, args.crop_box)
        if args.width or args.height or args.max_side:
            out = resize_image(
                out,
                width=args.width,
                height=args.height,
                max_side=args.max_side,
                keep_aspect=args.keep_aspect,
            )
        fmt = args.format or dest.suffix.lstrip(".")
        if out.mode == "RGBA" and normalize_format(fmt or "png") not in ("png", "webp"):
            fmt = "png"
            dest = dest.with_suffix(".png")
        save_image(out, dest, fmt=fmt, quality=args.quality)
        print(f"OK process {src} -> {dest}")
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if not getattr(args, "cmd", None):
        from .gui import run_gui

        run_gui()
        return 0

    if args._fn == "gui":
        from .gui import run_gui

        run_gui()
        return 0
    if args._fn == "resize":
        return _run_resize(args)
    if args._fn == "convert":
        return _run_convert(args)
    if args._fn == "dewatermark":
        return _run_dewatermark(args)
    if args._fn == "cutout":
        return _run_cutout(args)
    if args._fn == "crop":
        return _run_crop(args)
    if args._fn == "process":
        return _run_process(args)
    parser.print_help()
    return 2
