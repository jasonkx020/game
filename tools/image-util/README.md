# Image Util（本地图片工具）

全本地：缩放、改格式、**LaMa 去水印**、**BiRefNet 抠图**、**证件照制作**（公考/事业编/常见国标规格）、裁剪、**灰白去色**。  
GUI：**框选 → 右键/制作 → 右侧预览 → 应用结果**；**Ctrl+Z / Ctrl+Y** 撤销重做。

运行环境：**conda `game`**。

## 安装

```powershell
conda activate game
cd D:\workplace\source\GameS\tools\image-util

# CPU PyTorch（若尚未安装）
pip install torch torchvision --index-url https://download.pytorch.org/whl/cpu

pip install -r requirements.txt
```

首次 LaMa / BiRefNet 会下载权重，需可访问外网。

## GUI

```powershell
conda activate game
cd D:\workplace\source\GameS\tools\image-util
python -m image_util
```

### 工作流

1. **打开**图片  
2. 在左画布 **框选** 区域（松手不跑模型）  
3. **右键** 选择：去水印 / 抠图 / **证件照制作** / 灰白去色 / 裁剪为选区  
4. 成功后结果出现在 **右侧预览**（左侧工作图不变）  
5. 满意后点 **应用结果** 写入左侧工作图（可 Ctrl+Z 撤销该步）  
6. **Ctrl+Z / Ctrl+Y**；也可「重置结果」只清预览  
7. 顶栏 **保存 / 另存为**（优先导出预览，否则工作图）

### 工具说明

| 工具 | 操作 |
| :--- | :--- |
| 选择 | 画/调选区；右键执行功能 |
| 去水印 | 框选水印 → 右键或制作（LaMa） |
| 抠图 | 可选框主体 → 右键或制作 |
| 证件照制作 | 选规格 → 框选按比例校正 → 抠人换底缩放 |
| 裁剪 | 调框 / 框选后右键「裁剪为选区」 |
| 缩放 | 设 W/H/max-side → 制作 |
| 灰白去色 | 可选框、调强度 → 右键或制作 |
| 抓手 | 拖动画布；其它工具下 Ctrl+拖动 |

### 证件照规格（节选）

| 键 | 用途 | 像素 | 默认底 |
| :--- | :--- | :--- | :--- |
| `gwy` | 公务员/事业单位报名 | 295×413 | 蓝 |
| `1inch` / `2inch` / `small2` | 标准一寸/二寸/小二寸 | 见下表 | 蓝 |
| `1inch_large` | 大一寸 | 390×567 | 蓝 |
| `idcard` | 身份证常见电子 | 358×441 | 白 |
| `passport` | 护照/签证常用 | 390×567 | 白 |
| `social` / `driver` | 社保卡 / 驾驶证常见 | 358×441 / 260×378 | 蓝/白 |
| `keep` | 只换底 | 原尺寸 | 蓝 |

框选会按目标宽高比自动扩展再裁切。国考报名确认上传前，仍须用考录系统官方「照片处理工具」再处理。

## CLI

```powershell
python -m image_util idphoto --list-sizes
python -m image_util idphoto .\in.png -o .\gwy.jpg --size gwy --bg blue
python -m image_util idphoto .\in.png -o .\passport.jpg --size passport
python -m image_util crop .\in.png -o .\out.png --box 100,80,800,600
python -m image_util cutout .\in.png -o .\cut.png --mode birefnet
python -m image_util dewatermark .\in.png -o .\out.png --mode inpaint --box 900,1100,120,40
python -m image_util grayscale .\in.png -o .\gray.png --amount 1
```

未指定 `--bg` 时，证件照会使用该规格的默认底色。

## 说明

- 去水印：LaMa；抠图 / 证件照：BiRefNet（`ZhengPeng7/BiRefNet`）。  
- 证件照规格为常见电子像素参考，各地细则可能不同，以招考/受理机关最新要求为准。  
- 灰白去色：Pillow 降饱和度；有 `--box` 时仅处理选区。  
- 仅处理自有素材；CPU 上大图推理会较慢。
