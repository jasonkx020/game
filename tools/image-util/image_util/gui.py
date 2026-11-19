"""Photoshop-like GUI: pick a tool, adjust selection, then Make."""

from __future__ import annotations

import threading
from pathlib import Path
from tkinter import BooleanVar, Button, Canvas, DoubleVar, IntVar, Menu, StringVar, Tk, filedialog, messagebox, ttk

from PIL import Image, ImageTk

from .io_batch import load_image, save_image
from .ops_convert import normalize_format
from .ops_crop import crop_image
from .ops_cutout import cutout, parse_rgb
from .ops_dewatermark import Box, clamp_box, corner_box, dewatermark
from .ops_grayscale import grayscale_image
from .ops_idphoto import (
    ID_PHOTO_SPECS,
    expand_box_to_aspect,
    get_spec,
    make_id_photo,
    resolve_bg_rgb,
)
from .ops_resize import resize_image
from .preview import DisplayTransform, clamp_pan, composite_on_checker, fit_preview, overlay_box

_HANDLE = 8
_MAX_UNDO = 30
_MAKE_BG = "#3d7ea6"
_MAKE_BG_ACTIVE = "#2a5f80"
_MAKE_FG = "#ffffff"
_TOOL_LABELS = {
    "select": "选择",
    "dewatermark": "去水印",
    "cutout": "抠图",
    "idphoto": "证件照制作",
    "crop": "裁剪",
    "resize": "缩放",
    "grayscale": "灰白去色",
    "hand": "抓手",
}
class ImageUtilApp:
    def __init__(self, root: Tk) -> None:
        self.root = root
        root.title("Image Util — 框选右键制作 / Ctrl+Z 撤销")
        root.geometry("1280x860")
        root.minsize(1024, 700)

        self.src_path: Path | None = None
        self.src_img: Image.Image | None = None  # working image
        self.original_img: Image.Image | None = None
        self.result_img: Image.Image | None = None
        self.left_tf: DisplayTransform | None = None

        self.selection_box: Box | None = None
        self.crop_box: Box | None = None

        self._undo_stack: list[Image.Image] = []
        self._redo_stack: list[Image.Image] = []

        self._drag_start: tuple[float, float] | None = None
        self._photo_left: ImageTk.PhotoImage | None = None
        self._photo_right: ImageTk.PhotoImage | None = None
        self._rect_id: int | None = None

        self._box_edit: str | None = None
        self._box_edit_kind: str = "selection"  # selection | crop
        self._box_edit_origin: Box | None = None
        self._box_edit_mouse: tuple[float, float] | None = None
        self._syncing_crop_vars = False

        self.view_zoom = 1.0
        self.view_pan_x = 0.0
        self.view_pan_y = 0.0
        self._pan_start: tuple[float, float, float, float] | None = None
        self._panning = False
        self._preview_gen = 0
        self._preview_job = None
        self._pan_job = None
        self._busy = False

        self.tool = StringVar(value="select")
        self.zoom_label = StringVar(value="缩放 100%")
        self.left_hint = StringVar(value="")
        self.status = StringVar(value="请打开图片。框选后右键选功能，或点制作。Ctrl+Z 撤销。")

        self.corner = StringVar(value="br")
        self.margin_pct = DoubleVar(value=8.0)
        self.refine_inpaint = BooleanVar(value=True)
        self.cutout_mode = StringVar(value="birefnet")
        self.cutout_tol = IntVar(value=30)
        self.key_rgb = StringVar(value="255,255,255")
        self.crop_x = StringVar(value="0")
        self.crop_y = StringVar(value="0")
        self.crop_w = StringVar(value="0")
        self.crop_h = StringVar(value="0")
        self.width_var = StringVar(value="")
        self.height_var = StringVar(value="")
        self.max_side_var = StringVar(value="")
        self.keep_aspect = BooleanVar(value=True)
        self.gray_amount = DoubleVar(value=100.0)
        self.idphoto_bg = StringVar(value="blue")
        self.idphoto_custom_rgb = StringVar(value="67,142,219")
        self.idphoto_size = StringVar(value="gwy")
        self.idphoto_spec_hint = StringVar(value="")
        self.fmt = StringVar(value="png")
        self.quality = IntVar(value=90)

        self._tool_buttons: dict[str, ttk.Button] = {}
        self.options_frame: ttk.Frame | None = None
        self._make_btn: Button | None = None
        self._ctx_menu: Menu | None = None

        self._build()
        self.idphoto_size.trace_add("write", self._on_idphoto_size_changed)
        self._set_tool("select")
        self._try_enable_dnd()

    # ------------------------------------------------------------------ UI
    def _build(self) -> None:
        top = ttk.Frame(self.root, padding=8)
        top.pack(fill="x")
        ttk.Button(top, text="打开", command=self.open_file).pack(side="left", padx=2)
        ttk.Button(top, text="保存", command=self.save_overwrite).pack(side="left", padx=2)
        ttk.Button(top, text="另存为", command=self.save_as).pack(side="left", padx=2)
        ttk.Separator(top, orient="vertical").pack(side="left", fill="y", padx=8)
        ttk.Label(top, text="格式").pack(side="left")
        ttk.Combobox(top, textvariable=self.fmt, values=("png", "webp", "jpg", "bmp"), width=7, state="readonly").pack(
            side="left", padx=4
        )
        ttk.Label(top, text="质量").pack(side="left", padx=(6, 2))
        ttk.Scale(top, from_=40, to=100, variable=self.quality, orient="horizontal", length=100).pack(side="left")
        ttk.Button(top, text="适应窗口", command=lambda: self.reset_view()).pack(side="left", padx=8)
        ttk.Label(top, textvariable=self.zoom_label).pack(side="left", padx=6)
        ttk.Separator(top, orient="vertical").pack(side="left", fill="y", padx=8)
        ttk.Button(top, text="撤销", command=self.undo).pack(side="left", padx=2)
        ttk.Button(top, text="重做", command=self.redo).pack(side="left", padx=2)

        body = ttk.Frame(self.root)
        body.pack(fill="both", expand=True, padx=8, pady=4)

        tools = ttk.Frame(body, padding=(0, 0, 8, 0))
        tools.pack(side="left", fill="y")
        ttk.Label(tools, text="工具").pack(anchor="w", pady=(0, 4))
        for key, label in _TOOL_LABELS.items():
            btn = ttk.Button(tools, text=label, width=10, command=lambda k=key: self._set_tool(k))
            btn.pack(fill="x", pady=2)
            self._tool_buttons[key] = btn

        canvases = ttk.Panedwindow(body, orient="horizontal")
        canvases.pack(side="left", fill="both", expand=True)

        left_fr = ttk.Frame(canvases)
        right_fr = ttk.Frame(canvases)
        canvases.add(left_fr, weight=1)
        canvases.add(right_fr, weight=1)

        ttk.Label(left_fr, textvariable=self.left_hint).pack(anchor="w")
        self.left_canvas = Canvas(left_fr, bg="#2b2b2b", highlightthickness=0)
        self.left_canvas.pack(fill="both", expand=True)
        self.left_canvas.bind("<ButtonPress-1>", self._on_press)
        self.left_canvas.bind("<B1-Motion>", self._on_drag)
        self.left_canvas.bind("<ButtonRelease-1>", self._on_release)
        self.left_canvas.bind("<Button-3>", self._on_context_menu)
        self.left_canvas.bind("<Configure>", lambda _e: self._redraw_left())
        self._bind_view_nav(self.left_canvas)

        ttk.Label(right_fr, text="结果预览（透明以棋盘格显示）").pack(anchor="w")
        self.right_canvas = Canvas(right_fr, bg="#1e1e1e", highlightthickness=0)
        self.right_canvas.pack(fill="both", expand=True)
        self.right_canvas.bind("<ButtonPress-1>", self._on_result_pan_press)
        self.right_canvas.bind("<B1-Motion>", self._on_result_pan_drag)
        self.right_canvas.bind("<ButtonRelease-1>", self._on_result_pan_release)
        self.right_canvas.bind("<Configure>", lambda _e: self._redraw_right())
        self._bind_view_nav(self.right_canvas)

        bottom = ttk.Frame(self.root, padding=8)
        bottom.pack(fill="x")
        self.options_frame = ttk.LabelFrame(bottom, text="工具选项", padding=8)
        self.options_frame.pack(fill="x", pady=(0, 6))

        actions = ttk.Frame(bottom)
        actions.pack(fill="x")
        self._make_btn = Button(
            actions,
            text="制作",
            command=self.execute_tool,
            bg=_MAKE_BG,
            fg=_MAKE_FG,
            activebackground=_MAKE_BG_ACTIVE,
            activeforeground=_MAKE_FG,
            relief="raised",
            bd=2,
            padx=14,
            pady=4,
            font=("Microsoft YaHei UI", 10, "bold"),
            cursor="hand2",
        )
        self._make_btn.pack(side="left", padx=2)
        self._make_btn.bind("<ButtonPress-1>", self._on_make_press)
        self._make_btn.bind("<ButtonRelease-1>", self._on_make_release)
        self._make_btn.bind("<Leave>", self._on_make_leave)
        ttk.Button(actions, text="应用结果", command=self.apply_result_as_working).pack(side="left", padx=2)
        ttk.Button(actions, text="重置结果", command=self.reset_preview).pack(side="left", padx=2)
        ttk.Button(actions, text="清除选区", command=self.clear_box).pack(side="left", padx=2)
        ttk.Label(actions, textvariable=self.status).pack(side="left", padx=12)

        self._build_context_menu()
        self.root.bind_all("<Control-z>", self._on_undo_key)
        self.root.bind_all("<Control-Z>", self._on_undo_key)
        self.root.bind_all("<Control-y>", self._on_redo_key)
        self.root.bind_all("<Control-Y>", self._on_redo_key)

    def _on_make_press(self, _event=None) -> None:  # noqa: ANN001
        if self._make_btn is None or str(self._make_btn["state"]) == "disabled":
            return
        self._make_btn.configure(relief="sunken", bg=_MAKE_BG_ACTIVE)

    def _on_make_release(self, _event=None) -> None:  # noqa: ANN001
        self._reset_make_btn_style()

    def _on_make_leave(self, _event=None) -> None:  # noqa: ANN001
        self._reset_make_btn_style()

    def _reset_make_btn_style(self) -> None:
        if self._make_btn is None:
            return
        if str(self._make_btn["state"]) == "disabled":
            return
        self._make_btn.configure(relief="raised", bg=_MAKE_BG)

    def _set_make_busy(self, busy: bool) -> None:
        if self._make_btn is None:
            return
        if busy:
            self._make_btn.configure(state="disabled", relief="sunken", bg=_MAKE_BG_ACTIVE)
        else:
            self._make_btn.configure(state="normal", relief="raised", bg=_MAKE_BG)

    # ------------------------------------------------------------------ context menu
    def _build_context_menu(self) -> None:
        menu = Menu(self.root, tearoff=0)
        menu.add_command(label="去水印", command=lambda: self._run_from_menu("dewatermark"))
        menu.add_command(label="抠图", command=lambda: self._run_from_menu("cutout"))
        menu.add_command(label="证件照制作", command=lambda: self._run_from_menu("idphoto"))
        menu.add_command(label="灰白去色", command=lambda: self._run_from_menu("grayscale"))
        menu.add_command(label="裁剪为选区", command=lambda: self._run_from_menu("crop"))
        menu.add_separator()
        menu.add_command(label="清除选区", command=self.clear_box)
        menu.add_separator()
        menu.add_command(label="撤销  Ctrl+Z", command=self.undo)
        menu.add_command(label="重做  Ctrl+Y", command=self.redo)
        self._ctx_menu = menu

    def _on_context_menu(self, event) -> None:  # noqa: ANN001
        if self.src_img is None or self._ctx_menu is None:
            return
        if self._busy:
            return
        try:
            self._ctx_menu.tk_popup(event.x_root, event.y_root)
        finally:
            self._ctx_menu.grab_release()

    def _run_from_menu(self, tool: str) -> None:
        if self.src_img is None:
            return
        if self._busy:
            return
        if tool == "crop" and self.selection_box is None and self.crop_box is None:
            messagebox.showinfo("提示", "请先框选要裁剪的区域")
            return
        if tool == "crop" and self.selection_box is not None:
            self.crop_box = self.selection_box
            self._box_to_crop_vars()
        self._set_tool(tool)
        self._run_and_commit(tool)

    def _set_tool(self, tool: str) -> None:
        self.tool.set(tool)
        for k, btn in self._tool_buttons.items():
            try:
                btn.state(["!pressed"])
            except Exception:
                pass
            if k == tool:
                try:
                    btn.state(["pressed"])
                except Exception:
                    pass
        if tool == "crop" and self.crop_box is None and self.src_img is not None:
            self._init_default_crop(redraw=False)
        if tool == "cutout":
            self.fmt.set("png")
        if tool == "idphoto":
            self.fmt.set("jpg")
            self._snap_idphoto_selection(announce=True)
        self._rebuild_options()
        self._sync_hints()
        self._update_status_ready()
        self._redraw_left()

    def _clear_options(self) -> None:
        if self.options_frame is None:
            return
        for child in self.options_frame.winfo_children():
            child.destroy()

    def _rebuild_options(self) -> None:
        self._clear_options()
        assert self.options_frame is not None
        fr = self.options_frame
        tool = self.tool.get()
        fr.configure(text=f"工具选项 — {_TOOL_LABELS.get(tool, tool)}")

        if tool == "select":
            ttk.Label(
                fr, text="拖拽绘制选区；拖框内移动，拖边/角调整。框选后右键选功能；Ctrl+Z 撤销上一步。"
            ).pack(anchor="w")
        elif tool == "dewatermark":
            row = ttk.Frame(fr)
            row.pack(fill="x")
            ttk.Label(row, text="引擎: LaMa").pack(side="left")
            ttk.Checkbutton(row, text="精细修复", variable=self.refine_inpaint).pack(side="left", padx=12)
            ttk.Label(row, text="无选区时角落").pack(side="left", padx=(8, 2))
            ttk.Combobox(
                row, textvariable=self.corner, values=("br", "bl", "tr", "tl", "auto"), width=6, state="readonly"
            ).pack(side="left")
            ttk.Label(row, text="margin%").pack(side="left", padx=(8, 2))
            ttk.Scale(row, from_=1, to=25, variable=self.margin_pct, orient="horizontal", length=120).pack(side="left")
            ttk.Label(fr, text="先框选水印，再右键「去水印」或点「制作」。结果先出右侧预览，再点「应用结果」。").pack(
                anchor="w", pady=(4, 0)
            )
        elif tool == "cutout":
            row = ttk.Frame(fr)
            row.pack(fill="x")
            ttk.Label(row, text="方式").pack(side="left")
            ttk.Combobox(
                row, textvariable=self.cutout_mode, values=("birefnet", "flood", "colorkey"), width=10, state="readonly"
            ).pack(side="left", padx=4)
            ttk.Label(row, text="容差").pack(side="left", padx=(8, 2))
            ttk.Scale(row, from_=0, to=80, variable=self.cutout_tol, orient="horizontal", length=120).pack(side="left")
            ttk.Label(row, text="色键RGB").pack(side="left", padx=(8, 2))
            ttk.Entry(row, textvariable=self.key_rgb, width=12).pack(side="left")
            ttk.Label(fr, text="可选框选主体；右键「抠图」或点「制作」。结果先出右侧预览，再点「应用结果」。").pack(
                anchor="w", pady=(4, 0)
            )
        elif tool == "idphoto":
            row = ttk.Frame(fr)
            row.pack(fill="x")
            ttk.Label(row, text="底色").pack(side="left")
            for key, label in (("blue", "蓝"), ("red", "红"), ("white", "白"), ("custom", "自定义")):
                ttk.Radiobutton(row, text=label, value=key, variable=self.idphoto_bg).pack(side="left", padx=4)
            ttk.Label(row, text="RGB").pack(side="left", padx=(8, 2))
            ttk.Entry(row, textvariable=self.idphoto_custom_rgb, width=12).pack(side="left")
            row2 = ttk.Frame(fr)
            row2.pack(fill="x", pady=(4, 0))
            ttk.Label(row2, text="规格").pack(side="left")
            size_keys = tuple(ID_PHOTO_SPECS.keys())
            ttk.Combobox(
                row2,
                textvariable=self.idphoto_size,
                values=size_keys,
                width=14,
                state="readonly",
            ).pack(side="left", padx=4)
            ttk.Label(row2, textvariable=self.idphoto_spec_hint, wraplength=720).pack(side="left", padx=6)
            self._refresh_idphoto_spec_hint()
            ttk.Label(
                fr,
                text="含公考/事业编、一寸/二寸、身份证/护照等。框选按规格比例校正后制作。",
            ).pack(anchor="w", pady=(4, 0))
        elif tool == "crop":
            row = ttk.Frame(fr)
            row.pack(fill="x")
            for label, var in (("X", self.crop_x), ("Y", self.crop_y), ("W", self.crop_w), ("H", self.crop_h)):
                ttk.Label(row, text=label).pack(side="left", padx=(6, 2))
                ttk.Entry(row, textvariable=var, width=6).pack(side="left")
            ttk.Button(row, text="写入选区", command=self._crop_vars_to_box).pack(side="left", padx=6)
            ttk.Button(row, text="居中80%", command=lambda: self._init_default_crop()).pack(side="left", padx=2)
            ttk.Label(fr, text="调好裁剪框后右键「裁剪为选区」或点「制作」。").pack(anchor="w", pady=(4, 0))
        elif tool == "resize":
            row = ttk.Frame(fr)
            row.pack(fill="x")
            ttk.Label(row, text="W").pack(side="left")
            ttk.Entry(row, textvariable=self.width_var, width=7).pack(side="left", padx=2)
            ttk.Label(row, text="H").pack(side="left", padx=(6, 2))
            ttk.Entry(row, textvariable=self.height_var, width=7).pack(side="left", padx=2)
            ttk.Label(row, text="max-side").pack(side="left", padx=(6, 2))
            ttk.Entry(row, textvariable=self.max_side_var, width=7).pack(side="left", padx=2)
            ttk.Checkbutton(row, text="保持比例", variable=self.keep_aspect).pack(side="left", padx=8)
            ttk.Label(fr, text="设置尺寸后点「制作」。").pack(anchor="w", pady=(4, 0))
        elif tool == "grayscale":
            row = ttk.Frame(fr)
            row.pack(fill="x")
            ttk.Label(row, text="去色强度").pack(side="left")
            ttk.Scale(row, from_=0, to=100, variable=self.gray_amount, orient="horizontal", length=180).pack(
                side="left", padx=4
            )
            ttk.Label(fr, text="有选区则只处理选区；右键「灰白去色」或点「制作」。").pack(anchor="w", pady=(4, 0))
        elif tool == "hand":
            ttk.Label(fr, text="拖动画布平移；滚轮缩放。也可在其它工具下按住 Ctrl 拖动平移。").pack(anchor="w")

    def _sync_hints(self) -> None:
        tool = self.tool.get()
        hints = {
            "select": "工作图：框选后右键选功能；Ctrl+Z 撤销",
            "dewatermark": "工作图：框选水印 → 右键/制作（LaMa）",
            "cutout": "工作图：可选框主体 → 右键/制作（BiRefNet）",
            "idphoto": "工作图：框选人像（按证件照比例校正）→ 右键/制作",
            "crop": "工作图：调整裁剪框 → 右键/制作",
            "resize": "工作图：在选项里设尺寸 → 点制作",
            "grayscale": "工作图：可选框选区 → 右键/制作（灰白去色）",
            "hand": "拖动画布平移；滚轮缩放",
        }
        self.left_hint.set(hints.get(tool, ""))

    def _update_status_ready(self) -> None:
        tool = _TOOL_LABELS.get(self.tool.get(), self.tool.get())
        box = self.selection_box
        if self.tool.get() == "crop":
            box = self.crop_box
        if box is not None:
            x, y, w, h = box
            sel = f"{x},{y},{w},{h}"
        else:
            sel = "无"
        u, r = len(self._undo_stack), len(self._redo_stack)
        self.status.set(f"工具={tool} | 选区={sel} | 撤销{u}/重做{r} | 右键或制作")

    # ------------------------------------------------------------------ file / working image
    def _try_enable_dnd(self) -> None:
        try:
            from tkinterdnd2 import DND_FILES, TkinterDnD  # type: ignore

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
        self.original_img = img.copy()
        self.src_img = img
        self.result_img = None
        self.selection_box = None
        self.crop_box = None
        self._undo_stack.clear()
        self._redo_stack.clear()
        self._box_to_crop_vars()
        self.reset_view(redraw=False)
        self.status.set(f"已打开: {path.name} ({img.size[0]}x{img.size[1]})")
        self._redraw_both()
        self._update_status_ready()

    def clear_box(self) -> None:
        self.selection_box = None
        self.crop_box = None
        self._box_to_crop_vars()
        self._redraw_left()
        self._update_status_ready()

    def reset_preview(self) -> None:
        self.result_img = None
        self._redraw_right()
        self.status.set("已清空结果预览。")

    def apply_result_as_working(self) -> None:
        if self.result_img is None:
            messagebox.showinfo("提示", "没有可应用的结果")
            return
        if self.src_img is not None and self.result_img.size == self.src_img.size:
            # Already committed by auto-apply path — still allow explicit re-commit if bytes differ;
            # cheap size check + identity: if same object content after commit, inform user.
            pass
        self._commit_working(self.result_img, label="应用结果")

    def _export_image(self) -> Image.Image | None:
        if self.result_img is not None:
            return self.result_img
        return self.src_img

    def save_overwrite(self) -> None:
        if self.src_path is None:
            messagebox.showinfo("提示", "请先打开图片")
            return
        img = self._export_image()
        if img is None:
            return
        fmt = normalize_format(self.fmt.get())
        if img.mode == "RGBA" and fmt not in ("png", "webp"):
            fmt = "png"
            self.fmt.set("png")
        path = self.src_path.with_name(f"{self.src_path.stem}_out.{fmt}")
        self._save_to(path, img)

    def save_as(self) -> None:
        img = self._export_image()
        if img is None:
            messagebox.showinfo("提示", "请先打开图片")
            return
        fmt = normalize_format(self.fmt.get())
        if img.mode == "RGBA" and fmt not in ("png", "webp"):
            fmt = "png"
            self.fmt.set("png")
            messagebox.showinfo("提示", "含透明通道，已改为 PNG。")
        path = filedialog.asksaveasfilename(
            defaultextension=f".{fmt}",
            filetypes=[(fmt.upper(), f"*.{fmt}"), ("All", "*.*")],
        )
        if path:
            self._save_to(Path(path), img)

    def _save_to(self, path: Path, img: Image.Image) -> None:
        try:
            fmt = self.fmt.get()
            if img.mode == "RGBA" and normalize_format(fmt) not in ("png", "webp"):
                fmt = "png"
                path = path.with_suffix(".png")
            save_image(img, path, fmt=fmt, quality=self.quality.get())
            self.status.set(f"已保存: {path}")
        except Exception as e:  # noqa: BLE001
            messagebox.showerror("保存失败", str(e))

    # ------------------------------------------------------------------ undo / redo
    def _push_undo(self) -> None:
        if self.src_img is None:
            return
        self._undo_stack.append(self.src_img.copy())
        if len(self._undo_stack) > _MAX_UNDO:
            self._undo_stack.pop(0)
        self._redo_stack.clear()

    def _commit_working(self, img: Image.Image, *, label: str = "制作") -> None:
        if self.src_img is None:
            self.src_img = img.copy()
        else:
            # Skip push if identical size and tobytes match (already committed same pixels)
            same = (
                self.src_img.size == img.size
                and self.src_img.mode == img.mode
                and self.src_img.tobytes() == img.tobytes()
            )
            if same:
                self.result_img = img.copy()
                self._redraw_right()
                self.status.set(f"{label}：工作图未变化")
                return
            self._push_undo()
            self.src_img = img.copy()
        self.result_img = img.copy()
        self.selection_box = None
        self.crop_box = None
        self._box_to_crop_vars()
        self.reset_view(redraw=False)
        self._redraw_both()
        u = len(self._undo_stack)
        self.status.set(
            f"{label}完成 → {self.src_img.size[0]}x{self.src_img.size[1]}（已写入工作图，可 Ctrl+Z，撤销栈 {u}）"
        )

    def undo(self) -> None:
        if self._busy:
            return
        if not self._undo_stack or self.src_img is None:
            self.status.set("没有可撤销的步骤")
            return
        self._redo_stack.append(self.src_img.copy())
        self.src_img = self._undo_stack.pop()
        self.result_img = self.src_img.copy()
        self.selection_box = None
        self.crop_box = None
        self._box_to_crop_vars()
        self.reset_view(redraw=False)
        self._redraw_both()
        self.status.set(
            f"已撤销 → {self.src_img.size[0]}x{self.src_img.size[1]}（撤销{len(self._undo_stack)}/重做{len(self._redo_stack)}）"
        )

    def redo(self) -> None:
        if self._busy:
            return
        if not self._redo_stack or self.src_img is None:
            self.status.set("没有可重做的步骤")
            return
        self._undo_stack.append(self.src_img.copy())
        self.src_img = self._redo_stack.pop()
        self.result_img = self.src_img.copy()
        self.selection_box = None
        self.crop_box = None
        self._box_to_crop_vars()
        self.reset_view(redraw=False)
        self._redraw_both()
        self.status.set(
            f"已重做 → {self.src_img.size[0]}x{self.src_img.size[1]}（撤销{len(self._undo_stack)}/重做{len(self._redo_stack)}）"
        )

    def _on_undo_key(self, _event=None) -> str:  # noqa: ANN001
        self.undo()
        return "break"

    def _on_redo_key(self, _event=None) -> str:  # noqa: ANN001
        self.redo()
        return "break"

    # ------------------------------------------------------------------ execute / 制作
    def execute_tool(self) -> None:
        if self.src_img is None:
            messagebox.showinfo("提示", "请先打开图片")
            return
        if self._busy:
            return
        tool = self.tool.get()
        if tool in ("select", "hand"):
            messagebox.showinfo(
                "提示",
                f"「{_TOOL_LABELS[tool]}」工具无需制作，请右键选功能，或切换到去水印/抠图/证件照/裁剪/缩放/灰白去色",
            )
            return
        self._run_and_commit(tool)

    def _run_and_commit(self, tool: str) -> None:
        if self.src_img is None or self._busy:
            return
        if tool == "crop":
            if self.crop_box is None and self.selection_box is not None:
                self.crop_box = self.selection_box
                self._box_to_crop_vars()
            if self.crop_box is None:
                self._init_default_crop(redraw=False)
            if self.crop_box is None:
                return
        self._start_execute_worker(tool)

    def _start_execute_worker(self, tool: str | None = None) -> None:
        if self.src_img is None:
            return
        tool = tool or self.tool.get()
        self._preview_gen += 1
        gen = self._preview_gen
        src = self.src_img.copy()
        # Snapshot boxes for worker thread (crop may use selection)
        sel = self.selection_box
        crop = self.crop_box
        corner = self.corner.get()
        margin = f"{self.margin_pct.get()}%"
        refine = self.refine_inpaint.get()
        cutout_mode = self.cutout_mode.get()
        cutout_tol = int(self.cutout_tol.get())
        key_rgb = parse_rgb(self.key_rgb.get()) if tool == "cutout" else (255, 255, 255)
        gray_amount = float(self.gray_amount.get()) / 100.0
        width_s = self.width_var.get().strip()
        height_s = self.height_var.get().strip()
        max_side_s = self.max_side_var.get().strip()
        keep_aspect = self.keep_aspect.get()
        id_bg_choice = self.idphoto_bg.get()
        id_custom = self.idphoto_custom_rgb.get()
        id_size = self.idphoto_size.get()

        self._busy = True
        self._set_make_busy(True)
        self.status.set(f"制作中：{_TOOL_LABELS.get(tool, tool)}…")

        def worker() -> None:
            err: Exception | None = None
            out: Image.Image | None = None
            try:
                if tool == "dewatermark":
                    out = dewatermark(
                        src,
                        mode="inpaint",
                        box=sel,
                        corner=corner,  # type: ignore[arg-type]
                        margin=margin,
                        method="lama",
                        refine=refine,
                    )
                elif tool == "cutout":
                    out = cutout(
                        src,
                        mode=cutout_mode,  # type: ignore[arg-type]
                        box=sel,
                        tolerance=cutout_tol,
                        key_rgb=key_rgb,
                        lite=False,
                    )
                elif tool == "idphoto":
                    if id_bg_choice == "custom":
                        bg = resolve_bg_rgb(bg_rgb=parse_rgb(id_custom))
                    else:
                        bg = resolve_bg_rgb(id_bg_choice)
                    out = make_id_photo(src, bg_rgb=bg, size_key=id_size, box=sel)
                elif tool == "crop":
                    assert crop is not None
                    out = crop_image(src, crop)
                elif tool == "resize":
                    w = int(width_s) if width_s else None
                    h = int(height_s) if height_s else None
                    ms = int(max_side_s) if max_side_s else None
                    if w is None and h is None and ms is None:
                        raise ValueError("请设置 W / H / max-side 至少一个")
                    out = resize_image(src, width=w, height=h, max_side=ms, keep_aspect=keep_aspect)
                elif tool == "grayscale":
                    out = grayscale_image(src, amount=gray_amount, box=sel)
                else:
                    raise ValueError(f"工具 {tool} 不可制作")
            except Exception as e:  # noqa: BLE001
                err = e

            def done() -> None:
                self._busy = False
                self._set_make_busy(False)
                if gen != self._preview_gen:
                    return
                if err is not None:
                    messagebox.showerror("制作失败", str(err))
                    self._update_status_ready()
                    return
                assert out is not None
                self.result_img = out
                self._redraw_right()
                self.status.set(
                    f"预览就绪：{_TOOL_LABELS.get(tool, tool)} → {out.size[0]}x{out.size[1]}（点「应用结果」写入左侧）"
                )

            self.root.after(0, done)

        threading.Thread(target=worker, daemon=True).start()

    # ------------------------------------------------------------------ crop helpers
    def _init_default_crop(self, redraw: bool = True) -> None:
        if self.src_img is None:
            return
        ow, oh = self.src_img.size
        w = max(1, int(ow * 0.8))
        h = max(1, int(oh * 0.8))
        x = max(0, (ow - w) // 2)
        y = max(0, (oh - h) // 2)
        self.crop_box = clamp_box((x, y, w, h), ow, oh)
        self._box_to_crop_vars()
        if redraw:
            self._redraw_left()
        self._update_status_ready()

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
        self._box_to_crop_vars()
        self._redraw_left()
        self._update_status_ready()

    # ------------------------------------------------------------------ view nav
    @staticmethod
    def _ctrl_held(event) -> bool:  # noqa: ANN001
        return bool(event.state & 0x0004)

    def _bind_view_nav(self, canvas: Canvas) -> None:
        canvas.bind("<MouseWheel>", self._on_mousewheel)
        canvas.bind("<Button-4>", lambda e: self._on_mousewheel_linux(e, 1))
        canvas.bind("<Button-5>", lambda e: self._on_mousewheel_linux(e, -1))
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
        img = self.src_img
        if img is None and self.result_img is not None:
            img = self.result_img
        if img is None:
            return 1, 1, 1, 1
        canvas = self.left_canvas if self.src_img is not None else self.right_canvas
        return img.size[0], img.size[1], max(1, canvas.winfo_width()), max(1, canvas.winfo_height())

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
        ix = (event.x - ((cw - ow * old_scale) / 2.0 + self.view_pan_x)) / max(old_scale, 1e-6)
        iy = (event.y - ((ch - oh * old_scale) / 2.0 + self.view_pan_y)) / max(old_scale, 1e-6)
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

    # ------------------------------------------------------------------ box edit (selection / crop)
    def _active_edit_box(self) -> Box | None:
        if self._box_edit_kind == "crop":
            return self.crop_box
        return self.selection_box

    def _set_active_edit_box(self, box: Box | None) -> None:
        if self._box_edit_kind == "crop":
            self.crop_box = box
            self._box_to_crop_vars()
        else:
            self.selection_box = box

    def _box_canvas_rect(self, box: Box) -> tuple[float, float, float, float] | None:
        if self.left_tf is None:
            return None
        x, y, w, h = box
        x0, y0 = self.left_tf.image_to_canvas(x, y)
        x1, y1 = self.left_tf.image_to_canvas(x + w, y + h)
        return x0, y0, x1, y1

    def _handle_points(self, box: Box) -> dict[str, tuple[float, float]]:
        r = self._box_canvas_rect(box)
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

    def _hit_box_edit(self, cx: float, cy: float, box: Box | None) -> str | None:
        if box is None:
            return None
        for name, (hx, hy) in self._handle_points(box).items():
            if abs(cx - hx) <= _HANDLE and abs(cy - hy) <= _HANDLE:
                return name
        r = self._box_canvas_rect(box)
        if r is None:
            return None
        x0, y0, x1, y1 = r
        if min(x0, x1) <= cx <= max(x0, x1) and min(y0, y1) <= cy <= max(y0, y1):
            return "move"
        return None

    def _apply_box_edit(self, cx: float, cy: float) -> None:
        if self.src_img is None or self.left_tf is None or self._box_edit is None or self._box_edit_mouse is None:
            return
        mode = self._box_edit
        if mode != "new" and self._box_edit_origin is None:
            return
        ow, oh = self.src_img.size
        mx0, my0 = self._box_edit_mouse
        ix0, iy0 = self.left_tf.canvas_to_image(mx0, my0)
        ix1, iy1 = self.left_tf.canvas_to_image(cx, cy)
        dx, dy = ix1 - ix0, iy1 - iy0
        if mode == "new":
            x = min(ix0, ix1)
            y = min(iy0, iy1)
            w = max(1, abs(ix1 - ix0))
            h = max(1, abs(iy1 - iy0))
        else:
            assert self._box_edit_origin is not None
            ox, oy, bw, bh = self._box_edit_origin
            x, y, w, h = ox, oy, bw, bh
            if mode == "move":
                x, y = ox + dx, oy + dy
            else:
                x2, y2 = ox + bw, oy + bh
                if mode in ("w", "nw", "sw"):
                    x = ox + dx
                if mode in ("e", "ne", "se"):
                    x2 = ox + bw + dx
                if mode in ("n", "nw", "ne"):
                    y = oy + dy
                if mode in ("s", "sw", "se"):
                    y2 = oy + bh + dy
                if x2 < x:
                    x, x2 = x2, x
                if y2 < y:
                    y, y2 = y2, y
                w, h = x2 - x, y2 - y
        box = clamp_box((int(round(x)), int(round(y)), int(round(w)), int(round(h))), ow, oh)
        # Live aspect lock while editing under idphoto (except pure move).
        if (
            self.tool.get() == "idphoto"
            and self._box_edit_kind == "selection"
            and mode != "move"
        ):
            target = self._idphoto_target_wh()
            if target is not None:
                box = expand_box_to_aspect(box, ow, oh, target[0], target[1])
        self._set_active_edit_box(box)

    def _idphoto_target_wh(self) -> tuple[int, int] | None:
        try:
            return get_spec(self.idphoto_size.get()).pixels
        except ValueError:
            return None

    def _refresh_idphoto_spec_hint(self) -> None:
        try:
            spec = get_spec(self.idphoto_size.get())
        except ValueError:
            self.idphoto_spec_hint.set("")
            return
        if spec.pixels is None:
            px = "保持原尺寸"
        else:
            px = f"{spec.pixels[0]}×{spec.pixels[1]}"
        self.idphoto_spec_hint.set(f"{spec.label} | {px} | 默认{spec.default_bg}底 — {spec.note}")

    def _on_idphoto_size_changed(self, *_args) -> None:  # noqa: ANN001
        self._refresh_idphoto_spec_hint()
        try:
            spec = get_spec(self.idphoto_size.get())
        except ValueError:
            return
        # Apply recommended bg unless user chose custom RGB mode.
        if self.idphoto_bg.get() != "custom":
            self.idphoto_bg.set(spec.default_bg)
        if self.tool.get() != "idphoto":
            return
        self._snap_idphoto_selection(announce=True)
        self._redraw_left()
        self._update_status_ready()

    def _snap_idphoto_selection(self, *, announce: bool = False) -> None:
        if self.src_img is None or self.tool.get() != "idphoto":
            return
        target = self._idphoto_target_wh()
        if target is None:
            return
        ow, oh = self.src_img.size
        if self.selection_box is None:
            return
        before = self.selection_box
        self.selection_box = expand_box_to_aspect(before, ow, oh, target[0], target[1])
        if announce:
            try:
                label = get_spec(self.idphoto_size.get()).label
            except ValueError:
                label = self.idphoto_size.get()
            x, y, w, h = self.selection_box
            self.status.set(f"已按「{label}」比例调整选区 → {x},{y},{w},{h}")

    def _uses_selection_tool(self) -> bool:
        return self.tool.get() in ("select", "dewatermark", "cutout", "idphoto", "grayscale")

    def _uses_crop_tool(self) -> bool:
        return self.tool.get() == "crop"

    def _on_press(self, event) -> None:  # noqa: ANN001
        if self.src_img is None:
            return
        if self.tool.get() == "hand" or self._ctrl_held(event):
            self._box_edit = None
            self._drag_start = None
            self._on_pan_start(event)
            return
        self._panning = False

        if self._uses_crop_tool():
            self._box_edit_kind = "crop"
            hit = self._hit_box_edit(event.x, event.y, self.crop_box)
            self._box_edit = hit or "new"
            self._box_edit_origin = self.crop_box
            self._box_edit_mouse = (event.x, event.y)
            if self._box_edit == "new":
                self.crop_box = None
            return

        if self._uses_selection_tool():
            self._box_edit_kind = "selection"
            hit = self._hit_box_edit(event.x, event.y, self.selection_box)
            self._box_edit = hit or "new"
            self._box_edit_origin = self.selection_box
            self._box_edit_mouse = (event.x, event.y)
            if self._box_edit == "new":
                self.selection_box = None
            return

        # resize: no canvas edit
        self._box_edit = None

    def _on_drag(self, event) -> None:  # noqa: ANN001
        if self._panning or self._pan_start is not None:
            self._on_pan_drag(event)
            return
        if self._box_edit is not None:
            self._apply_box_edit(event.x, event.y)
            self._redraw_left()

    def _on_release(self, event) -> None:  # noqa: ANN001
        if self._panning or self._pan_start is not None:
            self._on_pan_end(event)
            return
        if self._box_edit is not None:
            self._apply_box_edit(event.x, event.y)
            self._box_edit = None
            self._box_edit_origin = None
            self._box_edit_mouse = None
            # Select-only: never auto-run models; idphoto snaps to aspect.
            if self.tool.get() == "idphoto":
                self._snap_idphoto_selection(announce=True)
            self._redraw_left()
            self._update_status_ready()
            return

    def _on_result_pan_press(self, event) -> None:  # noqa: ANN001
        if self.result_img is None:
            return
        self._on_pan_start(event)

    def _on_result_pan_drag(self, event) -> None:  # noqa: ANN001
        self._on_pan_drag(event)

    def _on_result_pan_release(self, event) -> None:  # noqa: ANN001
        self._on_pan_end(event)

    # ------------------------------------------------------------------ draw
    def _view_kwargs(self) -> dict:
        return {"zoom": self.view_zoom, "pan_x": self.view_pan_x, "pan_y": self.view_pan_y}

    def _draw_box_overlay(self, box: Box, *, color: str = "#ff6644", dim: bool = False) -> None:
        r = self._box_canvas_rect(box)
        if r is None:
            return
        x0, y0, x1, y1 = r
        if dim:
            cw = max(1, self.left_canvas.winfo_width())
            ch = max(1, self.left_canvas.winfo_height())
            dim_kw = {"fill": "#000000", "stipple": "gray50", "outline": "", "tags": "box_ui"}
            self.left_canvas.create_rectangle(0, 0, cw, min(y0, y1), **dim_kw)
            self.left_canvas.create_rectangle(0, max(y0, y1), cw, ch, **dim_kw)
            self.left_canvas.create_rectangle(0, min(y0, y1), min(x0, x1), max(y0, y1), **dim_kw)
            self.left_canvas.create_rectangle(max(x0, x1), min(y0, y1), cw, max(y0, y1), **dim_kw)
        self.left_canvas.create_rectangle(x0, y0, x1, y1, outline=color, width=2, dash=(6, 3), tags="box_ui")
        for hx, hy in self._handle_points(box).values():
            self.left_canvas.create_rectangle(
                hx - 4, hy - 4, hx + 4, hy + 4, fill=color, outline="#ffffff", width=1, tags="box_ui"
            )

    def _redraw_left(self) -> None:
        if self.src_img is None:
            self.left_canvas.delete("all")
            return
        cw = max(1, self.left_canvas.winfo_width())
        ch = max(1, self.left_canvas.winfo_height())
        preview, tf = fit_preview(self.src_img, max_side=4096, canvas_w=cw, canvas_h=ch, **self._view_kwargs())
        self.left_tf = tf

        # Soft image overlay for selection (non-crop) also keep corner hint when dewatermark & no box
        show_box = self.selection_box
        if show_box is None and self.tool.get() == "dewatermark":
            show_box = corner_box(
                *self.src_img.size,
                self.corner.get(),  # type: ignore[arg-type]
                f"{self.margin_pct.get()}%",
                self.src_img,
            )
            # only as guide — don't write into selection_box
            guide = show_box
        else:
            guide = None

        if self.selection_box is not None and not self._uses_crop_tool():
            x, y, w, h = self.selection_box
            sx = int(round(x * tf.scale))
            sy = int(round(y * tf.scale))
            sw = max(1, int(round(w * tf.scale)))
            sh = max(1, int(round(h * tf.scale)))
            preview = overlay_box(preview.convert("RGBA"), (sx, sy, sw, sh))
        elif guide is not None and self.tool.get() == "dewatermark":
            x, y, w, h = guide
            sx = int(round(x * tf.scale))
            sy = int(round(y * tf.scale))
            sw = max(1, int(round(w * tf.scale)))
            sh = max(1, int(round(h * tf.scale)))
            preview = overlay_box(
                preview.convert("RGBA"),
                (sx, sy, sw, sh),
                color=(255, 200, 64, 70),
                outline=(255, 180, 40, 200),
            )

        self._photo_left = ImageTk.PhotoImage(preview)
        self.left_canvas.delete("all")
        self.left_canvas.create_image(tf.offset_x, tf.offset_y, anchor="nw", image=self._photo_left)

        if self._uses_crop_tool() and self.crop_box is not None:
            self._draw_box_overlay(self.crop_box, color="#3ddc97", dim=True)
        elif self._uses_selection_tool() and self.selection_box is not None:
            self._draw_box_overlay(self.selection_box, color="#ff6644", dim=False)

    def _redraw_right(self) -> None:
        img = self.result_img
        self.right_canvas.delete("all")
        if img is None:
            return
        cw = max(1, self.right_canvas.winfo_width())
        ch = max(1, self.right_canvas.winfo_height())
        preview, tf = fit_preview(img, max_side=4096, canvas_w=cw, canvas_h=ch, **self._view_kwargs())
        if preview.mode in ("RGBA", "LA") or (preview.mode == "P" and "transparency" in preview.info):
            preview = composite_on_checker(preview)
        else:
            preview = preview.convert("RGB")
        self._photo_right = ImageTk.PhotoImage(preview)
        self.right_canvas.create_image(tf.offset_x, tf.offset_y, anchor="nw", image=self._photo_right)


def run_gui() -> None:
    root = Tk()
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
