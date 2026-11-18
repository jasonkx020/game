# Image Util（本地图片工具）

全本地：缩放、改格式、**LaMa 去水印**、**BiRefNet 抠图**。  
GUI 支持框选、滚轮缩放、Ctrl+拖动平移。

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

### 去水印（LaMa）

框选水印 → 松手预览；引擎为 LaMa（非 OpenCV）。

### 抠图（BiRefNet）

1. 勾选「抠图/透明底」，方式选 **birefnet**  
2. 可框选主体加速（只对 ROI 推理）；不框选则整图抠图  
3. 导出 **PNG** 保留透明  
4. 备选：`flood` / `colorkey`（无模型）

## CLI

```powershell
# BiRefNet 抠图
python -m image_util cutout .\in.png -o .\cut.png --mode birefnet
python -m image_util cutout .\in.png -o .\cut.png --mode birefnet --box 40,30,500,600

# LaMa 去水印
python -m image_util dewatermark .\in.png -o .\out.png --mode inpaint --box 900,1100,120,40
```

## 说明

- 去水印：LaMa；抠图：BiRefNet（`ZhengPeng7/BiRefNet`）。  
- 仅处理自有素材；CPU 上大图推理会较慢。
