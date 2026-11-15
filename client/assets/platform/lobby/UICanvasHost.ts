import {
  Camera,
  Canvas,
  Color,
  Layers,
  Node,
  UITransform,
  Widget,
  director,
} from 'cc'
import {
  LobbyTheme,
  UI_DESIGN_HEIGHT,
  UI_DESIGN_WIDTH,
  createUINode,
  paintSkyBackground,
} from './LobbyUIKit'
import { applyScreenFit, getVisibleFrameSize, onScreenFitResize } from './ScreenFit'

export { UI_DESIGN_WIDTH, UI_DESIGN_HEIGHT }

const tracked = new Set<Node>()

/**
 * 解析或创建 UI Canvas，并按 FIXED_HEIGHT 可视区域满屏铺开。
 */
export function resolveOrCreateUICanvas(from: Node): Node {
  applyScreenFit()

  let p: Node | null = from
  while (p) {
    if (isCanvasNode(p)) {
      ensureFullBleedCanvas(p)
      return p
    }
    for (const c of p.children) {
      if (isCanvasNode(c)) {
        ensureFullBleedCanvas(c)
        return c
      }
    }
    p = p.parent
  }

  const scene = director.getScene()
  if (scene) {
    const found = findCanvasInTree(scene)
    if (found) {
      ensureFullBleedCanvas(found)
      return found
    }
    return createUICanvas(scene)
  }

  return createUICanvas(from.parent ?? from)
}

function isCanvasNode(node: Node): boolean {
  return node.name === 'Canvas' || !!node.getComponent(Canvas)
}

function findCanvasInTree(root: Node): Node | null {
  if (isCanvasNode(root)) return root
  for (const child of root.children) {
    const hit = findCanvasInTree(child)
    if (hit) return hit
  }
  return null
}

function trackCanvas(canvasNode: Node): void {
  if (tracked.has(canvasNode)) return
  tracked.add(canvasNode)
  onScreenFitResize(() => {
    if (!canvasNode.isValid) {
      tracked.delete(canvasNode)
      return
    }
    ensureFullBleedCanvas(canvasNode)
  })
}

/** Canvas / 相机 / 满屏背景对齐当前可视尺寸，消除黑边 */
export function ensureFullBleedCanvas(canvasNode: Node): void {
  applyScreenFit()
  const frame = getVisibleFrameSize()
  const w = frame.width
  const h = frame.height

  let uit = canvasNode.getComponent(UITransform)
  if (!uit) uit = canvasNode.addComponent(UITransform)
  uit.setContentSize(w, h)

  let widget = canvasNode.getComponent(Widget)
  if (!widget) widget = canvasNode.addComponent(Widget)
  widget.isAlignTop = true
  widget.isAlignBottom = true
  widget.isAlignLeft = true
  widget.isAlignRight = true
  widget.top = 0
  widget.bottom = 0
  widget.left = 0
  widget.right = 0
  widget.alignMode = Widget.AlignMode.ON_WINDOW_RESIZE
  widget.updateAlignment()

  const canvas = canvasNode.getComponent(Canvas)
  if (canvas) canvas.alignCanvasWithScreen = true

  const cam =
    canvas?.cameraComponent ??
    canvasNode.getComponentInChildren(Camera)
  if (cam) {
    if (cam.projection === Camera.ProjectionType.ORTHO) {
      cam.orthoHeight = h / 2
    }
    cam.clearColor = LobbyTheme.bgDeep
  }

  ensureFullBleedSky(canvasNode, w, h)
  trackCanvas(canvasNode)
}

function ensureFullBleedSky(canvasNode: Node, w: number, h: number): void {
  let bg = canvasNode.getChildByName('FullBleedBg')
  const uitExisting = bg?.getComponent(UITransform)
  if (bg && uitExisting && Math.abs(uitExisting.contentSize.width - w) < 1 && Math.abs(uitExisting.contentSize.height - h) < 1) {
    if (bg.getSiblingIndex() !== 0) bg.setSiblingIndex(0)
    return
  }
  if (bg) bg.destroy()
  bg = createUINode('FullBleedBg')
  canvasNode.insertChild(bg, 0)
  bg.addComponent(UITransform).setContentSize(w, h)
  paintSkyBackground(bg, w, h)
}

function createUICanvas(parent: Node): Node {
  const frame = getVisibleFrameSize()
  const w = frame.width
  const h = frame.height

  const canvasNode = createUINode('Canvas')
  parent.addChild(canvasNode)

  const uit = canvasNode.addComponent(UITransform)
  uit.setContentSize(w, h)

  const widget = canvasNode.addComponent(Widget)
  widget.isAlignTop = true
  widget.isAlignBottom = true
  widget.isAlignLeft = true
  widget.isAlignRight = true
  widget.top = 0
  widget.bottom = 0
  widget.left = 0
  widget.right = 0
  widget.alignMode = Widget.AlignMode.ON_WINDOW_RESIZE

  const canvas = canvasNode.addComponent(Canvas)
  canvas.alignCanvasWithScreen = true

  const camNode = createUINode('UICamera')
  canvasNode.addChild(camNode)
  camNode.setPosition(0, 0, 1000)

  const camera = camNode.addComponent(Camera)
  camera.projection = Camera.ProjectionType.ORTHO
  camera.orthoHeight = h / 2
  camera.near = 0.1
  camera.far = 2000
  camera.visibility = Layers.Enum.UI_2D | Layers.Enum.UI_3D
  camera.priority = 10
  camera.clearFlags = 6 as never
  camera.clearColor = new Color(LobbyTheme.bgDeep.r, LobbyTheme.bgDeep.g, LobbyTheme.bgDeep.b, 255)

  canvas.cameraComponent = camera
  ensureFullBleedSky(canvasNode, w, h)
  widget.updateAlignment()
  trackCanvas(canvasNode)
  console.info('[UICanvasHost] runtime UI Canvas full-bleed', w, h)
  return canvasNode
}
