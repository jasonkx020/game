import {
  Camera,
  Canvas,
  Color,
  Layers,
  Node,
  UITransform,
  Widget,
  director,
  view,
} from 'cc'
import { UI_DESIGN_HEIGHT, UI_DESIGN_WIDTH, createUINode } from './LobbyUIKit'

export { UI_DESIGN_WIDTH, UI_DESIGN_HEIGHT }

/**
 * 解析或创建 UI Canvas。
 * - 代码 UI（Label/Graphics）必须挂在 Canvas/RenderRoot2D 下才会被收集渲染
 * - 节点 layer 必须为 UI_2D，才能被 UI 相机 visibility 命中
 * Lobby.scene 当前没有 Canvas，必须运行时创建，否则大厅 UI 不可见。
 */
export function resolveOrCreateUICanvas(from: Node): Node {
  let p: Node | null = from
  while (p) {
    if (isCanvasNode(p)) return p
    for (const c of p.children) {
      if (isCanvasNode(c)) return c
    }
    p = p.parent
  }

  const scene = director.getScene()
  if (scene) {
    const found = findCanvasInTree(scene)
    if (found) return found
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

function createUICanvas(parent: Node): Node {
  const design = view.getDesignResolutionSize()
  const w = design.width > 0 ? design.width : UI_DESIGN_WIDTH
  const h = design.height > 0 ? design.height : UI_DESIGN_HEIGHT

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
  // body #e8f4fd
  camera.clearColor = new Color(0xe8, 0xf4, 0xfd, 255)

  canvas.cameraComponent = camera
  console.info('[UICanvasHost] scene missing Canvas — created runtime UI Canvas', w, h)
  return canvasNode
}
