import {
  _decorator,
  Color,
  Component,
  director,
  EventTouch,
  Graphics,
  Label,
  Node,
  UITransform,
  Vec3,
} from 'cc'
import { liuzichongState } from './LiuzichongPushHandler'
import { SessionStore } from '../../scripts/SessionStore'
import { encodeProto } from '../../platform/sdk/protoHelpers'
import { MoveReq } from '../../platform/generated/pitaya/pitaya/liuzichong'
import { createLabelNode, createUINode, ensureUILayer, paintRoundRect, place } from '../../platform/lobby/LobbyUIKit'
import { resolveOrCreateUICanvas } from '../../platform/lobby/UICanvasHost'
import { GameHost } from '../../platform/host/GameHost'
import type { LobbyGameItem } from '../../platform/sdk/ApiClient'

const { ccclass, property } = _decorator

const SIZE = 4
const CELL = 78
const PAD = 40
const BOARD_PX = CELL * (SIZE - 1) + PAD * 2

const WOOD_BG = new Color(0xf2, 0xe3, 0xc4, 255)
const WOOD_LINE = new Color(0x5a, 0x3f, 0x2b, 255)
const WOOD_FRAME = new Color(0xb3, 0x92, 0x64, 255)
const PANEL_BG = new Color(0xcb, 0xb1, 0x8b, 255)
const TEXT_DARK = new Color(0x1f, 0x2e, 0x1b, 255)
const BTN_BG = new Color(0x5a, 0x3f, 0x2b, 255)
const BTN_TEXT = new Color(0xf0, 0xe3, 0xce, 255)
const SELECT_RING = new Color(0xdc, 0xa0, 0x28, 255)
const MOVE_DOT = new Color(0x64, 0xb4, 0x64, 180)
const OVERLAY_BG = new Color(0x3d, 0x2a, 0x1b, 220)

@ccclass('LiuzichongBoard')
export class LiuzichongBoard extends Component {
  @property(Label)
  statusLabel: Label | null = null

  @property(Label)
  overlayLabel: Label | null = null

  private gridGfx: Graphics | null = null
  private pieceGfx: Graphics | null = null
  private highlightGfx: Graphics | null = null
  private selected: { row: number; col: number } | null = null
  private unsub: (() => void) | null = null
  private roomIdLabel: Label | null = null
  private settleRoot: Node | null = null
  private settleLabel: Label | null = null
  private builtChrome = false

  start(): void {
    if (!SessionStore.session || !SessionStore.room || SessionStore.room.gameId !== 'liuzichong') {
      director.loadScene('Lobby')
      return
    }
    this.ensureUnderCanvas()
    if (SessionStore.mySeat >= 0) {
      liuzichongState.mySeat = SessionStore.mySeat
    }
    this.buildChrome()
    this.buildBoardNodes()
    this.unsub = liuzichongState.onChange(() => this.redraw())
    this.redraw()
  }

  /** 兜底：若 Board 不在 Canvas 子树，挪过去，否则整屏不可见 */
  private ensureUnderCanvas(): void {
    const canvas = resolveOrCreateUICanvas(this.node)
    let under = false
    for (let p: Node | null = this.node; p; p = p.parent) {
      if (p === canvas) {
        under = true
        break
      }
    }
    if (!under) {
      this.node.setParent(canvas)
      this.node.setPosition(0, 0, 0)
    }
    ensureUILayer(this.node)
  }

  onDestroy(): void {
    this.unsub?.()
  }

  private buildChrome(): void {
    if (this.builtChrome) return
    this.builtChrome = true
    const root = this.node

    const title = createLabelNode('Title', '六子冲', 28, TEXT_DARK, 280, 40)
    title.getComponent(Label)!.isBold = true
    title.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(title, 0, 420)
    root.addChild(title)

    const info = createUINode('InfoPanel')
    info.addComponent(UITransform).setContentSize(520, 56)
    paintRoundRect(info, 520, 56, PANEL_BG, 28)
    place(info, 0, 350)
    root.addChild(info)

    if (!this.statusLabel) {
      const statusNode = createLabelNode('Status', '准备中…', 18, TEXT_DARK, 360, 36)
      statusNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
      statusNode.getComponent(Label)!.isBold = true
      place(statusNode, -40, 0)
      info.addChild(statusNode)
      this.statusLabel = statusNode.getComponent(Label)
    }

    const backBtn = createUINode('BackBtn')
    backBtn.addComponent(UITransform).setContentSize(100, 40)
    paintRoundRect(backBtn, 100, 40, BTN_BG, 20)
    place(backBtn, 190, 0)
    info.addChild(backBtn)
    const backLbl = createLabelNode('BackLbl', '返回', 16, BTN_TEXT, 90, 32)
    backLbl.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(backLbl, 0, 0)
    backBtn.addChild(backLbl)
    backBtn.on(Node.EventType.TOUCH_END, () => this.onBackToHall())

    const roomHint = createLabelNode(
      'RoomId',
      `房号 ${SessionStore.room?.roomId?.slice(0, 8) ?? ''}…`,
      12,
      new Color(0x5a, 0x3f, 0x2b, 200),
      480,
      24,
    )
    roomHint.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(roomHint, 0, -380)
    root.addChild(roomHint)
    this.roomIdLabel = roomHint.getComponent(Label)

    const rule = createLabelNode(
      'Rule',
      '吃子：己-己-敌 / 敌-己-己，且敌方外侧为空（边界算空）',
      12,
      new Color(0x5a, 0x3f, 0x2b, 180),
      560,
      24,
    )
    rule.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(rule, 0, -410)
    root.addChild(rule)

    const settle = createUINode('SettleOverlay')
    settle.addComponent(UITransform).setContentSize(420, 280)
    paintRoundRect(settle, 420, 280, OVERLAY_BG, 24)
    place(settle, 0, 40)
    settle.active = false
    root.addChild(settle)
    this.settleRoot = settle
    const settleText = createLabelNode('SettleText', '对局结束', 22, BTN_TEXT, 360, 80)
    settleText.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    settleText.getComponent(Label)!.enableWrapText = true
    place(settleText, 0, 60)
    settle.addChild(settleText)
    this.settleLabel = settleText.getComponent(Label)
    this.overlayLabel = this.settleLabel

    const againBtn = createUINode('AgainBtn')
    againBtn.addComponent(UITransform).setContentSize(160, 44)
    paintRoundRect(againBtn, 160, 44, new Color(0x6e, 0x4f, 0x36, 255), 22)
    place(againBtn, -90, -60)
    settle.addChild(againBtn)
    const againLbl = createLabelNode('AgainLbl', '再来一局', 16, BTN_TEXT, 150, 36)
    againLbl.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(againLbl, 0, 0)
    againBtn.addChild(againLbl)
    againBtn.on(Node.EventType.TOUCH_END, () => void this.onRematch())

    const hallBtn = createUINode('HallBtn')
    hallBtn.addComponent(UITransform).setContentSize(160, 44)
    paintRoundRect(hallBtn, 160, 44, BTN_BG, 22)
    place(hallBtn, 90, -60)
    settle.addChild(hallBtn)
    const hallLbl = createLabelNode('HallLbl', '回大厅', 16, BTN_TEXT, 150, 36)
    hallLbl.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(hallLbl, 0, 0)
    hallBtn.addChild(hallLbl)
    hallBtn.on(Node.EventType.TOUCH_END, () => this.onBackToHall())

    ensureUILayer(root)
  }

  private buildBoardNodes(): void {
    const root = this.node
    let boardNode = root.getChildByName('Board')
    if (!boardNode) {
      boardNode = createUINode('Board')
      boardNode.setParent(root)
      boardNode.setPosition(new Vec3(0, 20, 0))
      const ui = boardNode.addComponent(UITransform)
      ui.setContentSize(BOARD_PX, BOARD_PX)
      boardNode.on(Node.EventType.TOUCH_END, this.onBoardTouch, this)
    }

    this.gridGfx = this.ensureGfx(boardNode, 'Grid', 0)
    this.pieceGfx = this.ensureGfx(boardNode, 'Pieces', 1)
    this.highlightGfx = this.ensureGfx(boardNode, 'Highlight', 2)
    this.drawGrid()
  }

  private ensureGfx(parent: Node, name: string, sibling: number): Graphics {
    let n = parent.getChildByName(name)
    if (!n) {
      n = createUINode(name)
      n.setParent(parent)
      const ui = n.addComponent(UITransform)
      ui.setContentSize(BOARD_PX, BOARD_PX)
    }
    n.setSiblingIndex(sibling)
    let g = n.getComponent(Graphics)
    if (!g) g = n.addComponent(Graphics)
    return g
  }

  private drawGrid(): void {
    const g = this.gridGfx
    if (!g) return
    g.clear()
    const half = BOARD_PX / 2
    g.fillColor = WOOD_BG
    g.roundRect(-half, -half, BOARD_PX, BOARD_PX, 24)
    g.fill()
    g.strokeColor = WOOD_FRAME
    g.lineWidth = 6
    g.roundRect(-half + 2, -half + 2, BOARD_PX - 4, BOARD_PX - 4, 22)
    g.stroke()

    g.strokeColor = WOOD_LINE
    g.lineWidth = 2.5
    const origin = -CELL * (SIZE - 1) / 2
    for (let i = 0; i < SIZE; i++) {
      const o = origin + i * CELL
      g.moveTo(origin, o)
      g.lineTo(origin + CELL * (SIZE - 1), o)
      g.moveTo(o, origin)
      g.lineTo(o, origin + CELL * (SIZE - 1))
    }
    g.stroke()
  }

  private redraw(): void {
    this.drawPieces()
    this.drawHighlights()
    this.updateStatus()
  }

  private drawPieces(): void {
    const g = this.pieceGfx
    if (!g) return
    g.clear()
    const board = liuzichongState.board
    const radius = CELL * 0.36
    for (let r = 0; r < SIZE; r++) {
      for (let c = 0; c < SIZE; c++) {
        const v = board[r * SIZE + c]
        if (v === 0) continue
        const { x, y } = this.cellCenter(r, c)
        if (v === 1) {
          g.fillColor = new Color(0x22, 0x22, 0x22, 255)
        } else {
          g.fillColor = new Color(0xf5, 0xf0, 0xe8, 255)
        }
        g.circle(x, y, radius)
        g.fill()
        if (v === 2) {
          g.strokeColor = new Color(0x50, 0x50, 0x50, 255)
          g.lineWidth = 2
          g.circle(x, y, radius)
          g.stroke()
        }
      }
    }
  }

  private drawHighlights(): void {
    const g = this.highlightGfx
    if (!g) return
    g.clear()
    if (liuzichongState.gameOver) return
    if (liuzichongState.mySeat < 0) return
    if (liuzichongState.currentSeat !== liuzichongState.mySeat) return

    if (this.selected) {
      const { x, y } = this.cellCenter(this.selected.row, this.selected.col)
      g.strokeColor = SELECT_RING
      g.lineWidth = 3
      g.circle(x, y, CELL * 0.42)
      g.stroke()
      for (const nb of this.neighbors(this.selected.row, this.selected.col)) {
        if (liuzichongState.board[nb.row * SIZE + nb.col] !== 0) continue
        const c = this.cellCenter(nb.row, nb.col)
        g.fillColor = MOVE_DOT
        g.circle(c.x, c.y, 10)
        g.fill()
      }
    }
  }

  private updateStatus(): void {
    if (this.roomIdLabel && SessionStore.room) {
      this.roomIdLabel.string = `房号 ${SessionStore.room.roomId}`
    }
    let text = '准备中…'
    if (liuzichongState.gameOver) {
      text =
        liuzichongState.winnerSeat === liuzichongState.mySeat
          ? '你赢了！'
          : liuzichongState.winnerSeat >= 0
            ? '你输了'
            : '对局结束'
    } else if (liuzichongState.mySeat < 0) {
      text = '等待座位信息…'
    } else if (liuzichongState.board.every((v) => v === 0)) {
      text = '等待开局（可把房号发给好友）…'
    } else if (liuzichongState.currentSeat === liuzichongState.mySeat) {
      text = liuzichongState.mySeat === 0 ? '轮到你了（黑）' : '轮到你了（白）'
    } else {
      text = '等待对手…'
    }
    if (this.statusLabel) this.statusLabel.string = text

    if (this.settleRoot) {
      this.settleRoot.active = liuzichongState.gameOver
      if (liuzichongState.gameOver && this.settleLabel) {
        this.settleLabel.string = text
      }
    }
  }

  private onBoardTouch(ev: EventTouch): void {
    if (liuzichongState.gameOver) return
    if (liuzichongState.mySeat < 0 || liuzichongState.currentSeat !== liuzichongState.mySeat) return
    const boardNode = this.node.getChildByName('Board')
    if (!boardNode) return
    const ui = boardNode.getComponent(UITransform)
    if (!ui) return
    const loc = ev.getUILocation()
    const world = new Vec3(loc.x, loc.y, 0)
    const local = ui.convertToNodeSpaceAR(world)
    const origin = -CELL * (SIZE - 1) / 2
    const col = Math.round((local.x - origin) / CELL)
    const row = Math.round((origin - local.y) / CELL + (SIZE - 1))
    if (row < 0 || row >= SIZE || col < 0 || col >= SIZE) return

    const myColor = liuzichongState.mySeat + 1
    const cell = liuzichongState.board[row * SIZE + col]

    if (this.selected && this.selected.row === row && this.selected.col === col) {
      this.selected = null
      this.redraw()
      return
    }

    if (cell === myColor) {
      this.selected = { row, col }
      this.redraw()
      return
    }

    if (this.selected && cell === 0) {
      const nb = this.neighbors(this.selected.row, this.selected.col)
      const isAdj = nb.some((p) => p.row === row && p.col === col)
      if (isAdj) {
        void this.sendMove(this.selected.row, this.selected.col, row, col)
        this.selected = null
      }
    }
  }

  private async sendMove(fromRow: number, fromCol: number, toRow: number, toCol: number): Promise<void> {
    const session = SessionStore.session
    const room = SessionStore.room
    if (!session || !room) return
    try {
      const body = encodeProto(MoveReq, {
        roomId: room.roomId,
        fromRow,
        fromCol,
        toRow,
        toCol,
      })
      await session.pitaya.request('game.liuzichong.move', body)
      SessionStore.appendLog(`[move] (${fromRow},${fromCol})->(${toRow},${toCol})`)
    } catch (e) {
      console.error('[Liuzichong] move failed', e)
      SessionStore.appendLog(`[error] move ${String(e)}`)
      this.redraw()
    }
  }

  private neighbors(row: number, col: number): Array<{ row: number; col: number }> {
    const out: Array<{ row: number; col: number }> = []
    const dirs = [
      [-1, 0],
      [1, 0],
      [0, -1],
      [0, 1],
    ]
    for (const [dr, dc] of dirs) {
      const nr = row + dr
      const nc = col + dc
      if (nr >= 0 && nr < SIZE && nc >= 0 && nc < SIZE) out.push({ row: nr, col: nc })
    }
    return out
  }

  private cellCenter(row: number, col: number): { x: number; y: number } {
    const origin = -CELL * (SIZE - 1) / 2
    const x = origin + col * CELL
    const y = origin + (SIZE - 1 - row) * CELL
    return { x, y }
  }

  private async onRematch(): Promise<void> {
    const gameId = 'liuzichong'
    SessionStore.resetRoom()
    SessionStore.session?.leave()
    SessionStore.session = null
    liuzichongState.reset()
    const game: LobbyGameItem = {
      gameId,
      name: '六子冲',
      minPlayers: 2,
      maxPlayers: 2,
      visible: true,
      pinned: false,
      sortOrder: 20,
    }
    try {
      await GameHost.launch({
        mode: 'lobby',
        gameId,
        game,
        fillBots: true,
        companionSessionId: SessionStore.companionSessionId ?? undefined,
      })
    } catch (e) {
      console.error('[Liuzichong] rematch', e)
      director.loadScene('Lobby')
    }
  }

  onBackToHall(): void {
    SessionStore.resetRoom()
    SessionStore.session?.leave()
    SessionStore.session = null
    liuzichongState.reset()
    director.loadScene('Lobby')
  }
}
