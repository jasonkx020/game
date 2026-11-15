import { ResolutionPolicy, Size, view } from 'cc'
import { UI_DESIGN_HEIGHT, UI_DESIGN_WIDTH } from './LobbyUIKit'

let applied = false
const resizeHandlers: Array<() => void> = []

/**
 * 竖屏棋牌：设计 720×1280 + FIXED_HEIGHT。
 * 高度铺满屏幕（无上下黑边），宽度随屏扩展；不拉伸变形。
 * 与 `.cursor/rules/cocos-ui.mdc` / LLM 提示词一致。
 */
export function applyScreenFit(): void {
  view.setDesignResolutionSize(UI_DESIGN_WIDTH, UI_DESIGN_HEIGHT, ResolutionPolicy.FIXED_HEIGHT)
  if (applied) return
  applied = true
  view.on('canvas-resize', onCanvasResize)
}

function onCanvasResize(): void {
  for (const h of resizeHandlers) {
    try {
      h()
    } catch (e) {
      console.warn('[ScreenFit] resize handler error', e)
    }
  }
}

/** 注册窗口变化回调（Canvas 满屏重算）；返回取消函数 */
export function onScreenFitResize(handler: () => void): () => void {
  applyScreenFit()
  resizeHandlers.push(handler)
  return () => {
    const i = resizeHandlers.indexOf(handler)
    if (i >= 0) resizeHandlers.splice(i, 1)
  }
}

/**
 * 当前可视逻辑尺寸（设计坐标系）。
 * FIXED_HEIGHT 下 height≈1280，宽屏 width>720 —— 用于满屏背景/Canvas。
 */
export function getVisibleFrameSize(): Size {
  applyScreenFit()
  const vis = view.getVisibleSize()
  const width = Math.max(UI_DESIGN_WIDTH, Math.ceil(vis.width || UI_DESIGN_WIDTH))
  const height = Math.max(UI_DESIGN_HEIGHT, Math.ceil(vis.height || UI_DESIGN_HEIGHT))
  return new Size(width, height)
}
