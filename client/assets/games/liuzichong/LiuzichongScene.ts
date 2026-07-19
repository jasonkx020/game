import { _decorator, Camera, Canvas, Color, Component, Node } from 'cc'
import { createUINode, ensureUILayer } from '../../platform/lobby/LobbyUIKit'
import { resolveOrCreateUICanvas } from '../../platform/lobby/UICanvasHost'
import { LiuzichongBoard } from './LiuzichongBoard'

const { ccclass } = _decorator

/** 场景入口：将棋盘 UI 挂到 Canvas 下（否则 Graphics/Label 不会被 UI 相机渲染） */
@ccclass('LiuzichongScene')
export class LiuzichongScene extends Component {
  start(): void {
    const canvas = resolveOrCreateUICanvas(this.node)
    this.tintCanvasBg(canvas)

    let root = canvas.getChildByName('BoardRoot')
    if (!root) {
      root = createUINode('BoardRoot')
      canvas.addChild(root)
    }
    ensureUILayer(root)

    if (!root.getComponent(LiuzichongBoard)) {
      root.addComponent(LiuzichongBoard)
    }
  }

  private tintCanvasBg(canvas: Node): void {
    const cam =
      canvas.getComponent(Canvas)?.cameraComponent ?? canvas.getComponentInChildren(Camera)
    if (cam) {
      // 暖色纸底，避免默认黑清屏看起来像「没画面」
      cam.clearColor = new Color(0xe8, 0xd8, 0xbc, 255)
    }
  }
}
