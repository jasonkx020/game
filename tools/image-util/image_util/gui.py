"""Tkinter GUI: preview, corner/box dewatermark, resize, convert."""

from __future__ import annotations

import threading
from pathlib import Path
from tkinter import BooleanVar, DoubleVar, IntVar, StringVar, Tk, filedialog, messagebox, ttk
from tkinter import Canvas

from PIL import Image, ImageTk

from .io_batch import load_image, save_image
from .ops_convert import normalize_format
from .ops_crop import crop_image
from .ops_cutout import cutout, parse_rgb
from .ops_dewatermark import Box, clamp_box, corner_box, dewatermark
from .ops_resize import resize_image
from .preview import DisplayTransform, clamp_pan, composite_on_checker, fit_preview, overlay_box

_HANDLE = 8  # canvas px hit size for crop handles
_HANDLE_NAMES = ("nw", "n", "ne", "e", "se", "s", "sw", "w")


class ImageUtilApp:
    def __init__(self, root: Tk) -> None:
        self.root = root
        root.title("Image Util — 缩放 / 转格式 / 去水印 / 抠图 / 裁剪")
        root.geometry("1180x860")
        root.minsize(960, 700)

        self.src_path: Path | None = None
        self.src_img: Image.Image | None = None
        self.result_img: Image.Image | None = None
        self.left_tf: DisplayTransform | None = None
        self.user_box: Box | None = None
        self.crop_box: Box | None = None
        self._drag_start: tuple[float, float] | None = None
        self._photo_left: ImageTk.PhotoImage | None = None
        self._photo_right: ImageTk.PhotoImage | None = None
        self._rect_id: int | None = None
        self._crop_edit: str | None = None  # move | nw|n|ne|... | new
        self._crop_edit_origin: Box | None = None
        self._crop_edit_mouse: tuple[float, float] | None = None
        self._syncing_crop_vars = False
        # Shared view for left/right comparison.
        self.view_zoom = 1.0
        self.view_pan_x = 0.0
        self.view_pan_y = 0.0
        self._pan_start: tuple[float, float, float, float] | None = None  # cx, cy, pan_x, pan_y
        self._panning = False
        self._preview_gen = 0
        self._preview_job = None
        self._pan_job = None
        self._busy = False
        self.zoom_label = StringVar(value="缩放 100%")

        self.mode = StringVar(value="inpaint")
        self.corner = StringVar(value="br")
        self.method = StringVar(value="lama")
        self.margin_pct = DoubleVar(value=8.0)
        self.fmt = StringVar(value="png")
        self.quality = IntVar(value=90)
        self.keep_aspect = BooleanVar(value=True)
        self.width_var = StringVar(value="")
        self.height_var = StringVar(value="")
        self.max_side_var = StringVar(value="")
        self.apply_dewatermark = BooleanVar(value=True)
        self.refine_inpaint = BooleanVar(value=True)
        self.apply_cutout = BooleanVar(value=False)
        self.apply_crop = BooleanVar(value=False)
        self.apply_resize = BooleanVar(value=False)
        self.cutout_mode = StringVar(value="birefnet")
        self.cutout_tol = IntVar(value=30)
        self.key_rgb = StringVar(value="255,255,255")
        self.crop_x = StringVar(value="0")
        self.crop_y = StringVar(value="0")
        self.crop_w = StringVar(value="0")
        self.crop_h = StringVar(value="0")
        # What a mouse drag means: watermark / cutout subject / crop frame.
        self.select_purpose = StringVar(value="dewatermark")

        self._build()
        self._try_enable_dnd()

    def _build(self) -> None:
        top = ttk.Frame(self.root, padding=8)
        top.pack(fill="x")
        ttk.Button(top, text="打开", command=self.open_file).pack(side="left", padx=2)
        ttk.Button(top, text="保存", command=self.save_overwrite).pack(side="left", padx=2)
        ttk.Button(top, text="另存为", command=self.save_as).pack(side="left", padx=2)
        ttk.Button(top, text="清除选区", command=self.clear_box).pack(side="left", padx=2)
        ttk.Button(top, text="应用裁剪", command=self.apply_crop_now).pack(side="left", padx=2)
        ttk.Button(top, text="选区去水印", command=self.dewatermark_selection).pack(side="left", padx=8)
        ttk.Button(top, text="应用预览", command=lambda: self.apply_preview(quick=False)).pack(side="left", padx=2)
        ttk.Button(top, text="重置结果", command=self.reset_preview).pack(side="left", padx=2)
        ttk.Button(top, text="适应窗口", command=lambda: self.reset_view()).pack(side="left", padx=8)
        ttk.Label(top, textvariable=self.zoom_label).pack(side="left", padx=6)

        paned = ttk.Panedwindow(self.root, orient="horizontal")
        paned.pack(fill="both", expand=True, padx=8, pady=4)

        left_fr = ttk.Frame(paned)
        right_fr = ttk.Frame(paned)
        paned.add(left_fr, weight=1)
        paned.add(right_fr, weight=1)

        self.left_hint = StringVar(value="原图：拖拽框选水印；滚轮缩放；Ctrl+拖动平移（可看四角）")
        ttk.Label(left_fr, textvariable=self.left_hint).pack(anchor="w")
        self.left_canvas = Canvas(left_fr, bg="#2b2b2b", highlightthickness=0)
        self.left_canvas.pack(fill="both", expand=True)
        self.left_canvas.bind("<ButtonPress-1>", self._on_press)
        self.left_canvas.bind("<B1-Motion>", self._on_drag)
        self.left_canvas.bind("<ButtonRelease-1>", self._on_release)
        self.left_canvas.bind("<Configure>", lambda _e: self._redraw_left())
        self._bind_view_nav(self.left_canvas)

        ttk.Label(right_fr, text="处理后预览（滚轮缩放；Ctrl+拖动平移；透明棋盘格）").pack(anchor="w")
        self.right_canvas = Canvas(right_fr, bg="#1e1e1e", highlightthickness=0)
        self.right_canvas.pack(fill="both", expand=True)
        self.right_canvas.bind("<ButtonPress-1>", self._on_right_press)
        self.right_canvas.bind("<B1-Motion>", self._on_right_drag)
        self.right_canvas.bind("<ButtonRelease-1>", self._on_right_release)
        self.right_canvas.bind("<Configure>", lambda _e: self._redraw_right())
        self._bind_view_nav(self.right_canvas)

        ctrl = ttk.LabelFrame(self.root, text="参数", padding=8)
        ctrl.pack(fill="x", padx=8, pady=8)

        row0 = ttk.Frame(ctrl)
        row0.pack(fill="x", pady=2)
        ttk.Label(row0, text="框选用途").pack(side="left")
        ttk.Radiobutton(
            row0, text="去水印(选水印)", variable=self.select_purpose, value="dewatermark", command=self._sync_hints
        ).pack(side="left", padx=4)
        ttk.Radiobutton(
            row0, text="抠图(选主体)", variable=self.select_purpose, value="cutout", command=self._sync_hints
        ).pack(side="left", padx=4)
        ttk.Radiobutton(
            row0, text="裁剪(调框)", variable=self.select_purpose, value="crop", command=self._on_crop_purpose
        ).pack(side="left", padx=4)

        row1 = ttk.Frame(ctrl)
        row1.pack(fill="x", pady=2)
        ttk.Checkbutton(row1, text="去水印", variable=self.apply_dewatermark, command=self._sync_hints).pack(side="left")
        ttk.Label(row1, text="mode").pack(side="left", padx=(8, 2))
        ttk.Combobox(row1, textvariable=self.mode, values=("inpaint", "crop"), width=10, state="readonly").pack(
            side="left"
        )
        ttk.Label(row1, text="corner").pack(side="left", padx=(8, 2))
        ttk.Combobox(
            row1, textvariable=self.corner, values=("br", "bl", "tr", "tl", "auto"), width=8, state="readonly"
        ).pack(side="left")
        ttk.Label(row1, text="引擎").pack(side="left", padx=(8, 2))
        ttk.Label(row1, text="LaMa").pack(side="left")
        ttk.Checkbutton(row1, text="精细修复(更大上下文)", variable=self.refine_inpaint).pack(side="left", padx=8)
        ttk.Label(row1, text="margin%").pack(side="left", padx=(8, 2))
        ttk.Scale(row1, from_=1, to=25, variable=self.margin_pct, orient="horizontal", length=120).pack(side="left")

        row_cut = ttk.Frame(ctrl)
        row_cut.pack(fill="x", pady=2)
        ttk.Checkbutton(row_cut, text="抠图/透明底", variable=self.apply_cutout, command=self._on_cutout_toggle).pack(
            side="left"
        )
        ttk.Label(row_cut, text="方式").pack(side="left", padx=(8, 2))
        ttk.Combobox(
            row_cut,
            textvariable=self.cutout_mode,
            values=("birefnet", "flood", "colorkey"),
            width=10,
            state="readonly",
        ).pack(side="left")
        ttk.Label(row_cut, text="容差").pack(side="left", padx=(8, 2))
        ttk.Scale(row_cut, from_=0, to=80, variable=self.cutout_tol, orient="horizontal", length=120).pack(side="left")
        ttk.Label(row_cut, text="色键RGB").pack(side="left", padx=(8, 2))
        ttk.Entry(row_cut, textvariable=self.key_rgb, width=12).pack(side="left")
        ttk.Label(row_cut, text="(BiRefNet 可框选主体加速)").pack(side="left", padx=8)

        row_crop = ttk.Frame(ctrl)
        row_crop.pack(fill="x", pady=2)
        ttk.Checkbutton(row_crop, text="裁剪", variable=self.apply_crop, command=self._on_crop_toggle).pack(side="left")
        ttk.Label(row_crop, text="X").pack(side="left", padx=(8, 2))
        ttk.Entry(row_crop, textvariable=self.crop_x, width=6).pack(side="left")
        ttk.Label(row_crop, text="Y").pack(side="left", padx=(4, 2))
        ttk.Entry(row_crop, textvariable=self.crop_y, width=6).pack(side="left")
        ttk.Label(row_crop, text="W").pack(side="left", padx=(4, 2))
        ttk.Entry(row_crop, textvariable=self.crop_w, width=6).pack(side="left")
        ttk.Label(row_crop, text="H").pack(side="left", padx=(4, 2))
        ttk.Entry(row_crop, textvariable=self.crop_h, width=6).pack(side="left")
        ttk.Button(row_crop, text="写入选区", command=self._crop_vars_to_box).pack(side="left", padx=6)
        ttk.Button(row_crop, text="居中80%", command=self._init_default_crop).pack(side="left", padx=2)
        ttk.Label(row_crop, text="拖边/角调大小，拖框内移动").pack(side="left", padx=8)

        row2 = ttk.Frame(ctrl)
        row2.pack(fill="x", pady=2)
        ttk.Checkbutton(row2, text="缩放", variable=self.apply_resize).pack(side="left")
        ttk.Label(row2, text="W").pack(side="left", padx=(8, 2))
        ttk.Entry(row2, textvariable=self.width_var, width=7).pack(side="left")
        ttk.Label(row2, text="H").pack(side="left", padx=(8, 2))
        ttk.Entry(row2, textvariable=self.height_var, width=7).pack(side="left")
        ttk.Label(row2, text="max-side").pack(side="left", padx=(8, 2))
        ttk.Entry(row2, textvariable=self.max_side_var, width=7).pack(side="left")
        ttk.Checkbutton(row2, text="保持比例", variable=self.keep_aspect).pack(side="left", padx=8)

        row3 = ttk.Frame(ctrl)
        row3.pack(fill="x", pady=2)
        ttk.Label(row3, text="格式").pack(side="left")
        ttk.Combobox(row3, textvariable=self.fmt, values=("png", "webp", "jpg", "bmp"), width=8, state="readonly").pack(
            side="left", padx=4
        )
        ttk.Label(row3, text="quality").pack(side="left", padx=(8, 2))
        ttk.Scale(row3, from_=40, to=100, variable=self.quality, orient="horizontal", length=160).pack(side="left")
        self.status = StringVar(value="请打开图片。默认：左侧拖拽框选水印 → 松手自动去水印。")
        ttk.Label(ctrl, textvariable=self.status).pack(anchor="w", pady=(6, 0))
        self._sync_hints()

    def _try_enable_dnd(self) -> None:
        try:
            from tkinterdnd2 import DND_FILES, TkinterDnD  # type: ignore

            # Only works if root was created as TkinterDnD.Tk; soft-fail otherwise.
            self.left_canvas.drop_target_register(DND_FILES)  # type: ignore[attr-defined]
            self.left_canvas.dnd_bind("<<Drop>>", self._on_drop)  # type: ignore[attr-defined]
            _ = TkinterDnD
        except Exception:
            pass

    def _on_drop(self, event) -> None:  # noqa: ANN001
        path = str(event.data).strip().strip("{}")
        if path:
            self.load_path(Path(path))

    def open_file(self) -> None:
        path = filedialog.askopenfilename(
            title="打开图片",
            filetypes=[
                ("Images", "*.png;*.jpg;*.jpeg;*.webp;*.bmp;*.tif;*.tiff"),
                ("All", "*.*"),
            ],
        )
        if path:
            self.load_path(Path(path))

    def load_path(self, path: Path) -> None:
        try:
            img = load_image(path)
        except Exception as e:  # noqa: BLE001
            messagebox.showerror("打开失败", str(e))
            return
        self.src_path = path
        self.src_img = img
        self.user_box = None
        self.crop_box = None
        self.result_img = None
        self._box_to_crop_vars()
        self.reset_view(redraw=False)
        self.status.set(f"已打开: {path.name} ({img.size[0]}x{img.size[1]})")
        self._redraw_left()
        self.apply_preview()

    def clear_box(self) -> None:
        self.user_box = None
        if self.select_purpose.get() == "crop":
            self.crop_box = None
            self.apply_crop.set(False)
            self._box_to_crop_vars()
        self._redraw_left()
        self.status.set("已清除框选。")

    def reset_preview(self) -> None:
        self.result_img = self.src_img.copy() if self.src_img else None
        self._redraw_right()
        self.status.set("已重置为原图。")

    def _on_cutout_toggle(self) -> None:
        if self.apply_cutout.get():
            self.fmt.set("png")
            self.select_purpose.set("cutout")
        self._sync_hints()

    def _on_crop_toggle(self) -> None:
        if self.apply_crop.get():
            self.select_purpose.set("crop")
            if self.crop_box is None and self.src_img is not None:
                self._init_default_crop()
            self._sync_hints()
            self._redraw_left()
            self.apply_preview(quick=True)
        else:
            self._sync_hints()

    def _on_crop_purpose(self) -> None:
        self.apply_crop.set(True)
        if self.crop_box is None and self.src_img is not None:
            self._init_default_crop()
        self._sync_hints()
        self._redraw_left()

    def _init_default_crop(self) -> None:
        if self.src_img is None:
            return
        ow, oh = self.src_img.size
        w = max(1, int(ow * 0.8))
        h = max(1, int(oh * 0.8))
        x = max(0, (ow - w) // 2)
        y = max(0, (oh - h) // 2)
        self.crop_box = clamp_box((x, y, w, h), ow, oh)
        self.apply_crop.set(True)
        self._box_to_crop_vars()
        self._redraw_left()
        self.status.set(f"裁剪框：x={x},y={y},w={w},h={h}（可拖动/调大小）")

    def _box_to_crop_vars(self) -> None:
        self._syncing_crop_vars = True
        try:
            if self.crop_box is None:
                self.crop_x.set("0")
                self.crop_y.set("0")
                self.crop_w.set("0")
                self.crop_h.set("0")
            else:
                x, y, w, h = self.crop_box
                self.crop_x.set(str(x))
                self.crop_y.set(str(y))
                self.crop_w.set(str(w))
                self.crop_h.set(str(h))
        finally:
            self._syncing_crop_vars = False

    def _crop_vars_to_box(self) -> None:
        if self.src_img is None or self._syncing_crop_vars:
            return
        try:
            x = int(self.crop_x.get().strip() or "0")
            y = int(self.crop_y.get().strip() or "0")
            w = int(self.crop_w.get().strip() or "0")
            h = int(self.crop_h.get().strip() or "0")
        except ValueError:
            messagebox.showinfo("提示", "裁剪 X/Y/W/H 需为整数")
            return
        self.crop_box = clamp_box((x, y, w, h), *self.src_img.size)
        self.apply_crop.set(True)
        self._box_to_crop_vars()
        self._redraw_left()
        self.apply_preview(quick=True)

    def apply_crop_now(self) -> None:
        if self.src_img is None:
            messagebox.showinfo("提示", "请先打开图片")
            return
        if self.crop_box is None:
            self._init_default_crop()
        self.apply_crop.set(True)
        self.select_purpose.set("crop")
        self.apply_preview(quick=True)

    def _sync_hints(self) -> None:
        purpose = self.select_purpose.get()
        if purpose == "cutout":
            self.left_hint.set("原图：BiRefNet 抠图（可框选主体加速）；滚轮缩放；Ctrl+拖动平移")
        elif purpose == "crop":
            self.left_hint.set("原图：拖出裁剪框；拖框内移动，拖边/角调大小；Ctrl+拖动平移")
        else:
            self.left_hint.set("原图：框选水印去水印；滚轮缩放；Ctrl+拖动平移查看边角")

    @staticmethod
    def _ctrl_held(event) -> bool:  # noqa: ANN001
        return bool(event.state & 0x0004)

    def _bind_view_nav(self, canvas: Canvas) -> None:
        canvas.bind("<MouseWheel>", self._on_mousewheel)
        canvas.bind("<Button-4>", lambda e: self._on_mousewheel_linux(e, 1))
        canvas.bind("<Button-5>", lambda e: self._on_mousewheel_linux(e, -1))
        # Arrow keys pan when canvas focused.
        canvas.bind("<Left>", lambda e: self._nudge_pan(20, 0))
        canvas.bind("<Right>", lambda e: self._nudge_pan(-20, 0))
        canvas.bind("<Up>", lambda e: self._nudge_pan(0, 20))
        canvas.bind("<Down>", lambda e: self._nudge_pan(0, -20))
        canvas.bind("<Enter>", lambda e: canvas.focus_set())

    def reset_view(self, redraw: bool = True) -> None:
        self.view_zoom = 1.0
        self.view_pan_x = 0.0
        self.view_pan_y = 0.0
        self.zoom_label.set("缩放 100%")
        if redraw:
            self._redraw_both()

    def _redraw_both(self) -> None:
        self._redraw_left()
        self._redraw_right()

    def _pan_ref_size(self) -> tuple[int, int, int, int]:
        """Reference image/canvas size for shared pan limits (prefer left view)."""
        img = self.src_img
        if img is None and self.result_img is not None:
            img = self.result_img
        if img is None:
            return 1, 1, 1, 1
        canvas = self.left_canvas if self.src_img is not None else self.right_canvas
        cw = max(1, canvas.winfo_width())
        ch = max(1, canvas.winfo_height())
        return img.size[0], img.size[1], cw, ch

    def _clamp_view_pan(self) -> None:
        ow, oh, cw, ch = self._pan_ref_size()
        self.view_pan_x, self.view_pan_y = clamp_pan(
            self.view_pan_x,
            self.view_pan_y,
            orig_w=ow,
            orig_h=oh,
            canvas_w=cw,
            canvas_h=ch,
            zoom=self.view_zoom,
        )

    def _nudge_pan(self, dx: float, dy: float) -> None:
        if self.src_img is None and self.result_img is None:
            return
        self.view_pan_x += dx
        self.view_pan_y += dy
        self._clamp_view_pan()
        self._redraw_both()

    def _on_mousewheel_linux(self, event, direction: int) -> None:  # noqa: ANN001
        event.delta = 120 * direction
        self._on_mousewheel(event)

    def _on_mousewheel(self, event) -> None:  # noqa: ANN001
        if self.src_img is None and self.result_img is None:
            return
        # Zoom toward cursor; sync left/right for comparison.
        factor = 1.12 if event.delta > 0 else (1 / 1.12)
        old_zoom = self.view_zoom
        new_zoom = max(0.25, min(16.0, old_zoom * factor))
        if abs(new_zoom - old_zoom) < 1e-6:
            return

        canvas = event.widget
        cw = max(1, canvas.winfo_width())
        ch = max(1, canvas.winfo_height())
        img = self.src_img if self.src_img is not None else self.result_img
        assert img is not None
        ow, oh = img.size
        base = min(cw / float(ow), ch / float(oh))
        old_scale = base * old_zoom
        new_scale = base * new_zoom

        # Image pixel under cursor before zoom.
        ix = (event.x - ((cw - ow * old_scale) / 2.0 + self.view_pan_x)) / max(old_scale, 1e-6)
        iy = (event.y - ((ch - oh * old_scale) / 2.0 + self.view_pan_y)) / max(old_scale, 1e-6)
        # Keep that point under cursor after zoom.
        self.view_pan_x = event.x - (cw - ow * new_scale) / 2.0 - ix * new_scale
        self.view_pan_y = event.y - (ch - oh * new_scale) / 2.0 - iy * new_scale
        self.view_zoom = new_zoom
        self._clamp_view_pan()
        self.zoom_label.set(f"缩放 {int(round(new_zoom * 100))}%")
        self._redraw_both()

    def _on_pan_start(self, event) -> None:  # noqa: ANN001
        self._panning = True
        self._pan_start = (event.x, event.y, self.view_pan_x, self.view_pan_y)
        try:
            event.widget.config(cursor="fleur")
        except Exception:
            pass

    def _on_pan_drag(self, event) -> None:  # noqa: ANN001
        if self._pan_start is None:
            return
        x0, y0, px, py = self._pan_start
        self.view_pan_x = px + (event.x - x0)
        self.view_pan_y = py + (event.y - y0)
        self._clamp_view_pan()
        # Throttle redraw while dragging to keep pan smooth.
        if self._pan_job is None:
            self._pan_job = self.root.after(16, self._flush_pan_redraw)

    def _flush_pan_redraw(self) -> None:
        self._pan_job = None
        self._redraw_both()

    def _on_pan_end(self, event) -> None:  # noqa: ANN001
        self._pan_start = None
        self._panning = False
        try:
            event.widget.config(cursor="")
        except Exception:
            pass

    def dewatermark_selection(self) -> None:
        """Force inpaint on current box (or prompt to draw one)."""
        if self.src_img is None:
            messagebox.showinfo("提示", "请先打开图片")
            return
        if self.user_box is None:
            messagebox.showinfo("提示", "请先在左侧拖拽框选水印区域")
            return
        self.select_purpose.set("dewatermark")
        self.apply_dewatermark.set(True)
        self.mode.set("inpaint")
        self._sync_hints()
        self.apply_preview(quick=False)
        self.status.set(f"已对框选区域去水印 {self.user_box}")

    def _active_box(self) -> Box | None:
        if self.src_img is None:
            return None
        if self.user_box is not None:
            return self.user_box
        if self.apply_dewatermark.get() and not self.apply_cutout.get():
            return corner_box(
                *self.src_img.size,
                self.corner.get(),  # type: ignore[arg-type]
                f"{self.margin_pct.get()}%",
                self.src_img,
            )
        return None

    def _process(self, img: Image.Image, *, preview: bool = False) -> Image.Image:
        out = img
        if self.apply_dewatermark.get():
            # When both cutout+dewatermark: box is shared; prefer user box for both.
            box = self.user_box
            # Interactive preview: skip heavy refine for snappy box-select UX.
            refine = False if preview else self.refine_inpaint.get()
            out = dewatermark(
                out,
                mode=self.mode.get(),  # type: ignore[arg-type]
                box=box,
                corner=self.corner.get(),  # type: ignore[arg-type]
                margin=f"{self.margin_pct.get()}%",
                method=self.method.get(),  # type: ignore[arg-type]
                refine=refine,
            )
        if self.apply_cutout.get():
            key = parse_rgb(self.key_rgb.get())
            out = cutout(
                out,
                mode=self.cutout_mode.get(),  # type: ignore[arg-type]
                box=self.user_box,
                tolerance=int(self.cutout_tol.get()),
                key_rgb=key,
                lite=bool(preview),
            )
        if self.apply_crop.get() and self.crop_box is not None:
            # If earlier ops changed size (e.g. dewatermark crop mode), clamp to current.
            out = crop_image(out, self.crop_box)
        if self.apply_resize.get():
            w = int(self.width_var.get()) if self.width_var.get().strip() else None
            h = int(self.height_var.get()) if self.height_var.get().strip() else None
            ms = int(self.max_side_var.get()) if self.max_side_var.get().strip() else None
            out = resize_image(out, width=w, height=h, max_side=ms, keep_aspect=self.keep_aspect.get())
        return out

    def apply_preview(self, *, debounce_ms: int = 0, quick: bool = True) -> None:
        """Schedule preview on a worker thread so the UI stays responsive.

        quick=True: fast inpaint for box-select; quick=False: honor「精细修复」.
        """
        if self.src_img is None:
            return
        if self._preview_job is not None:
            try:
                self.root.after_cancel(self._preview_job)
            except Exception:
                pass
            self._preview_job = None

        def _start() -> None:
            self._preview_job = None
            self._start_preview_worker(quick=quick)

        if debounce_ms > 0:
            self._preview_job = self.root.after(debounce_ms, _start)
        else:
            _start()

    def _start_preview_worker(self, *, quick: bool) -> None:
        if self.src_img is None:
            return
        self._preview_gen += 1
        gen = self._preview_gen
        src = self.src_img.copy()
        self._busy = True
        self.status.set("LaMa 快速预览中…" if quick else "LaMa 精细处理中（首次会加载模型）…")

        def worker() -> None:
            err: Exception | None = None
            out: Image.Image | None = None
            try:
                out = self._process(src, preview=quick)
            except Exception as e:  # noqa: BLE001
                err = e

            def done() -> None:
                self._busy = False
                if gen != self._preview_gen:
                    return
                if err is not None:
                    messagebox.showerror("处理失败", str(err))
                    return
                assert out is not None
                self.result_img = out
                self._redraw_left()
                self._redraw_right()
                parts: list[str] = []
                if self.apply_dewatermark.get():
                    parts.append(
                        f"去水印:{'框选' if self.user_box else self.corner.get()}/{self.mode.get()}"
                        + ("·快预览" if quick else "")
                    )
                if self.apply_cutout.get():
                    parts.append(f"抠图:{self.cutout_mode.get()}")
                if self.apply_crop.get() and self.crop_box is not None:
                    parts.append(f"裁剪:{self.crop_box[2]}x{self.crop_box[3]}")
                if self.apply_resize.get():
                    parts.append("缩放")
                tag = " + ".join(parts) if parts else "原图"
                self.status.set(f"预览完成（{tag}）→ {out.size[0]}x{out.size[1]}")

            self.root.after(0, done)

        threading.Thread(target=worker, daemon=True).start()

    def save_overwrite(self) -> None:
        if self.src_path is None:
            messagebox.showinfo("提示", "请先打开图片")
            return
        fmt = normalize_format(self.fmt.get())
        if self.apply_cutout.get() and fmt not in ("png", "webp"):
            fmt = "png"
            self.fmt.set("png")
        self._save_to(self.src_path.with_name(f"{self.src_path.stem}_out.{fmt}"))

    def save_as(self) -> None:
        fmt = normalize_format(self.fmt.get())
        if self.apply_cutout.get() and fmt not in ("png", "webp"):
            fmt = "png"
            self.fmt.set("png")
            messagebox.showinfo("提示", "抠图含透明通道，已改为 PNG 导出。")
        path = filedialog.asksaveasfilename(
            defaultextension=f".{fmt}",
            filetypes=[(fmt.upper(), f"*.{fmt}"), ("All", "*.*")],
        )
        if path:
            self._save_to(Path(path))

    def _save_to(self, path: Path) -> None:
        if self.src_img is None:
            return
        self.status.set("保存处理中…")
        self.root.update_idletasks()
        try:
            # Save uses full refine quality (ROI-accelerated).
            out = self._process(self.src_img, preview=False)
            fmt = self.fmt.get()
            if out.mode == "RGBA" and normalize_format(fmt) not in ("png", "webp"):
                fmt = "png"
                path = path.with_suffix(".png")
            save_image(out, path, fmt=fmt, quality=self.quality.get())
            self.result_img = out
            self._redraw_right()
            self.status.set(f"已保存: {path}")
        except Exception as e:  # noqa: BLE001
            messagebox.showerror("保存失败", str(e))

    def _crop_canvas_rect(self) -> tuple[float, float, float, float] | None:
        if self.crop_box is None or self.left_tf is None:
            return None
        x, y, w, h = self.crop_box
        tf = self.left_tf
        x0, y0 = tf.image_to_canvas(x, y)
        x1, y1 = tf.image_to_canvas(x + w, y + h)
        return x0, y0, x1, y1

    def _crop_handle_points(self) -> dict[str, tuple[float, float]]:
        r = self._crop_canvas_rect()
        if r is None:
            return {}
        x0, y0, x1, y1 = r
        mx, my = (x0 + x1) / 2, (y0 + y1) / 2
        return {
            "nw": (x0, y0),
            "n": (mx, y0),
            "ne": (x1, y0),
            "e": (x1, my),
            "se": (x1, y1),
            "s": (mx, y1),
            "sw": (x0, y1),
            "w": (x0, my),
        }

    def _hit_crop_edit(self, cx: float, cy: float) -> str | None:
        """Return handle name, 'move', or None."""
        handles = self._crop_handle_points()
        for name, (hx, hy) in handles.items():
            if abs(cx - hx) <= _HANDLE and abs(cy - hy) <= _HANDLE:
                return name
        r = self._crop_canvas_rect()
        if r is None:
            return None
        x0, y0, x1, y1 = r
        if min(x0, x1) <= cx <= max(x0, x1) and min(y0, y1) <= cy <= max(y0, y1):
            return "move"
        return None

    def _apply_crop_edit(self, cx: float, cy: float) -> None:
        if self.src_img is None or self.left_tf is None or self._crop_edit is None or self._crop_edit_mouse is None:
            return
        mode = self._crop_edit
        if mode != "new" and self._crop_edit_origin is None:
            return
        ow, oh = self.src_img.size
        mx0, my0 = self._crop_edit_mouse
        ix0, iy0 = self.left_tf.canvas_to_image(mx0, my0)
        ix1, iy1 = self.left_tf.canvas_to_image(cx, cy)
        dx, dy = ix1 - ix0, iy1 - iy0
        if mode == "new":
            x = min(ix0, ix1)
            y = min(iy0, iy1)
            w = max(1, abs(ix1 - ix0))
            h = max(1, abs(iy1 - iy0))
        else:
            assert self._crop_edit_origin is not None
            ox, oy, ow_box, oh_box = self._crop_edit_origin
            x, y, w, h = ox, oy, ow_box, oh_box
            if mode == "move":
                x, y = ox + dx, oy + dy
            else:
                x2, y2 = ox + ow_box, oy + oh_box
                if mode in ("w", "nw", "sw"):
                    x = ox + dx
                if mode in ("e", "ne", "se"):
                    x2 = ox + ow_box + dx
                if mode in ("n", "nw", "ne"):
                    y = oy + dy
                if mode in ("s", "sw", "se"):
                    y2 = oy + oh_box + dy
                if x2 < x:
                    x, x2 = x2, x
                if y2 < y:
                    y, y2 = y2, y
                w, h = x2 - x, y2 - y
        self.crop_box = clamp_box((int(round(x)), int(round(y)), int(round(w)), int(round(h))), ow, oh)
        self._box_to_crop_vars()

    def _on_press(self, event) -> None:  # noqa: ANN001
        if self.src_img is None:
            return
        # Ctrl + drag = pan (so corners remain reachable when zoomed).
        if self._ctrl_held(event):
            self._drag_start = None
            self._crop_edit = None
            self._on_pan_start(event)
            return
        self._panning = False

        if self.select_purpose.get() == "crop":
            hit = self._hit_crop_edit(event.x, event.y) if self.crop_box is not None else None
            self._crop_edit = hit or "new"
            self._crop_edit_origin = self.crop_box
            self._crop_edit_mouse = (event.x, event.y)
            self._drag_start = (event.x, event.y)
            if self._crop_edit == "new":
                self.crop_box = None
            self.apply_crop.set(True)
            return

        self._crop_edit = None
        self._drag_start = (event.x, event.y)
        if self._rect_id is not None:
            self.left_canvas.delete(self._rect_id)
            self._rect_id = None

    def _on_drag(self, event) -> None:  # noqa: ANN001
        if self._panning or self._pan_start is not None:
            self._on_pan_drag(event)
            return
        if self.select_purpose.get() == "crop" and self._crop_edit is not None:
            self._apply_crop_edit(event.x, event.y)
            self._redraw_left()
            return
        if self._drag_start is None:
            return
        x0, y0 = self._drag_start
        if self._rect_id is not None:
            self.left_canvas.delete(self._rect_id)
        self._rect_id = self.left_canvas.create_rectangle(
            x0, y0, event.x, event.y, outline="#ff4444", width=2, dash=(4, 2)
        )

    def _on_release(self, event) -> None:  # noqa: ANN001
        if self._panning or self._pan_start is not None:
            self._on_pan_end(event)
            return

        if self.select_purpose.get() == "crop" and self._crop_edit is not None:
            self._apply_crop_edit(event.x, event.y)
            self._crop_edit = None
            self._crop_edit_origin = None
            self._crop_edit_mouse = None
            self._drag_start = None
            if self.crop_box is not None:
                x, y, w, h = self.crop_box
                self.status.set(f"裁剪框：x={x},y={y},w={w},h={h}")
                self.apply_preview(debounce_ms=30, quick=True)
            self._redraw_left()
            return

        if self._drag_start is None or self.left_tf is None or self.src_img is None:
            self._drag_start = None
            return
        x0, y0 = self._drag_start
        self._drag_start = None
        if abs(event.x - x0) < 3 and abs(event.y - y0) < 3:
            return
        self.user_box = self.left_tf.canvas_box_to_image(x0, y0, event.x, event.y)

        # Box-select → dewatermark is the default action.
        if self.select_purpose.get() == "dewatermark":
            self.apply_dewatermark.set(True)
            self.mode.set("inpaint")
        elif self.select_purpose.get() == "cutout":
            self.apply_cutout.set(True)
            self.cutout_mode.set("birefnet")
            self.fmt.set("png")

        # Debounce + background fast preview: avoid UI freeze on box release.
        self.apply_preview(debounce_ms=50, quick=True)
        if self.select_purpose.get() == "dewatermark" and self.user_box is not None:
            x, y, w, h = self.user_box
            self.status.set(f"框选已提交：x={x},y={y},w={w},h={h}（快速预览中…）")

    def _on_right_press(self, event) -> None:  # noqa: ANN001
        # Right panel: Ctrl+drag or plain drag pans for inspection.
        if self.result_img is None:
            return
        self._on_pan_start(event)

    def _on_right_drag(self, event) -> None:  # noqa: ANN001
        self._on_pan_drag(event)

    def _on_right_release(self, event) -> None:  # noqa: ANN001
        self._on_pan_end(event)

    def _view_kwargs(self) -> dict:
        return {
            "zoom": self.view_zoom,
            "pan_x": self.view_pan_x,
            "pan_y": self.view_pan_y,
        }

    def _draw_crop_overlay(self) -> None:
        """Draw crop rectangle + handles in canvas coordinates."""
        r = self._crop_canvas_rect()
        if r is None:
            return
        x0, y0, x1, y1 = r
        # Dim outside first, then border/handles on top.
        cw = max(1, self.left_canvas.winfo_width())
        ch = max(1, self.left_canvas.winfo_height())
        dim = {"fill": "#000000", "stipple": "gray50", "outline": "", "tags": "crop_ui"}
        self.left_canvas.create_rectangle(0, 0, cw, min(y0, y1), **dim)
        self.left_canvas.create_rectangle(0, max(y0, y1), cw, ch, **dim)
        self.left_canvas.create_rectangle(0, min(y0, y1), min(x0, x1), max(y0, y1), **dim)
        self.left_canvas.create_rectangle(max(x0, x1), min(y0, y1), cw, max(y0, y1), **dim)
        self.left_canvas.create_rectangle(
            x0, y0, x1, y1, outline="#3ddc97", width=2, dash=(6, 3), tags="crop_ui"
        )
        for hx, hy in self._crop_handle_points().values():
            self.left_canvas.create_rectangle(
                hx - 4,
                hy - 4,
                hx + 4,
                hy + 4,
                fill="#3ddc97",
                outline="#ffffff",
                width=1,
                tags="crop_ui",
            )

    def _redraw_left(self) -> None:
        if self.src_img is None:
            self.left_canvas.delete("all")
            return
        cw = max(1, self.left_canvas.winfo_width())
        ch = max(1, self.left_canvas.winfo_height())
        preview, tf = fit_preview(
            self.src_img, max_side=4096, canvas_w=cw, canvas_h=ch, **self._view_kwargs()
        )
        self.left_tf = tf
        # Watermark / cutout selection overlay (not crop).
        if self.select_purpose.get() != "crop":
            box = self._active_box()
            if box is not None:
                x, y, w, h = box
                sx = int(round(x * tf.scale))
                sy = int(round(y * tf.scale))
                sw = max(1, int(round(w * tf.scale)))
                sh = max(1, int(round(h * tf.scale)))
                preview = overlay_box(preview.convert("RGBA"), (sx, sy, sw, sh))
        self._photo_left = ImageTk.PhotoImage(preview)
        self.left_canvas.delete("all")
        self.left_canvas.create_image(tf.offset_x, tf.offset_y, anchor="nw", image=self._photo_left)
        self._rect_id = None
        if self.select_purpose.get() == "crop" or self.apply_crop.get():
            self._draw_crop_overlay()

    def _redraw_right(self) -> None:
        img = self.result_img
        self.right_canvas.delete("all")
        if img is None:
            return
        cw = max(1, self.right_canvas.winfo_width())
        ch = max(1, self.right_canvas.winfo_height())
        # Use source size for pan math sync when result size matches; otherwise fit result alone
        # but still share zoom/pan so comparison stays aligned when sizes equal.
        preview, tf = fit_preview(img, max_side=4096, canvas_w=cw, canvas_h=ch, **self._view_kwargs())
        if preview.mode in ("RGBA", "LA") or (preview.mode == "P" and "transparency" in preview.info):
            preview = composite_on_checker(preview)
        else:
            preview = preview.convert("RGB")
        self._photo_right = ImageTk.PhotoImage(preview)
        self.right_canvas.create_image(tf.offset_x, tf.offset_y, anchor="nw", image=self._photo_right)


def run_gui() -> None:
    root = Tk()
    # Prefer a theme that looks ok on Windows.
    try:
        style = ttk.Style(root)
        if "vista" in style.theme_names():
            style.theme_use("vista")
        elif "clam" in style.theme_names():
            style.theme_use("clam")
    except Exception:
        pass
    ImageUtilApp(root)
    root.mainloop()
