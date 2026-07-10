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

const { ccclass, property } = _decorator

const SIZE = 4
const CELL = 72
const PAD = 36

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

  start(): void {
    if (!SessionStore.session || !SessionStore.room || SessionStore.room.gameId !== 'liuzichong') {
      director.loadScene('Lobby')
      return
    }
    this.buildBoardNodes()
    this.unsub = liuzichongState.onChange(() => this.redraw())
    this.redraw()
  }

  onDestroy(): void {
    this.unsub?.()
  }

  private buildBoardNodes(): void {
    const root = this.node
    let boardNode = root.getChildByName('Board')
    if (!boardNode) {
      boardNode = new Node('Board')
      boardNode.setParent(root)
      boardNode.setPosition(new Vec3(0, 0, 0))
      const ui = boardNode.addComponent(UITransform)
      ui.setContentSize(CELL * SIZE + PAD * 2, CELL * SIZE + PAD * 2)
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
      n = new Node(name)
      n.setParent(parent)
      const ui = n.addComponent(UITransform)
      ui.setContentSize(CELL * SIZE + PAD * 2, CELL * SIZE + PAD * 2)
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
    g.fillColor = new Color(242, 227, 196, 255)
    g.rect(-(CELL * SIZE + PAD * 2) / 2, -(CELL * SIZE + PAD * 2) / 2, CELL * SIZE + PAD * 2, CELL * SIZE + PAD * 2)
    g.fill()
    g.strokeColor = new Color(120, 90, 50, 255)
    g.lineWidth = 2
    for (let i = 0; i <= SIZE; i++) {
      const o = -CELL * SIZE / 2 + i * CELL
      g.moveTo(-CELL * SIZE / 2, o)
      g.lineTo(CELL * SIZE / 2, o)
      g.moveTo(o, -CELL * SIZE / 2)
      g.lineTo(o, CELL * SIZE / 2)
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
    for (let r = 0; r < SIZE; r++) {
      for (let c = 0; c < SIZE; c++) {
        const v = board[r * SIZE + c]
        if (v === 0) continue
        const { x, y } = this.cellCenter(r, c)
        const radius = CELL * 0.32
        if (v === 1) {
          g.fillColor = new Color(35, 35, 35, 255)
        } else {
          g.fillColor = new Color(245, 240, 232, 255)
          g.strokeColor = new Color(80, 80, 80, 255)
          g.lineWidth = 2
        }
        g.circle(x, y, radius)
        g.fill()
        if (v === 2) {
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
      g.strokeColor = new Color(220, 160, 40, 255)
      g.lineWidth = 3
      g.circle(x, y, CELL * 0.38)
      g.stroke()
      for (const nb of this.neighbors(this.selected.row, this.selected.col)) {
        if (liuzichongState.board[nb.row * SIZE + nb.col] !== 0) continue
        const c = this.cellCenter(nb.row, nb.col)
        g.fillColor = new Color(100, 180, 100, 80)
        g.rect(c.x - CELL * 0.35, c.y - CELL * 0.35, CELL * 0.7, CELL * 0.7)
        g.fill()
      }
    }
  }

  private updateStatus(): void {
    if (this.statusLabel) {
      if (liuzichongState.gameOver) {
        const win =
          liuzichongState.winnerSeat === liuzichongState.mySeat
            ? '你赢了！'
            : liuzichongState.winnerSeat >= 0
              ? '你输了'
              : '对局结束'
        this.statusLabel.string = win
      } else if (liuzichongState.mySeat < 0) {
        this.statusLabel.string = '等待座位信息…'
      } else if (liuzichongState.currentSeat === liuzichongState.mySeat) {
        this.statusLabel.string = liuzichongState.mySeat === 0 ? '轮到你了（黑）' : '轮到你了（白）'
      } else {
        this.statusLabel.string = '等待对手…'
      }
    }
    if (this.overlayLabel) {
      this.overlayLabel.node.active = liuzichongState.gameOver
      if (liuzichongState.gameOver) {
        this.overlayLabel.string = this.statusLabel?.string ?? '对局结束'
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
    const col = Math.floor((local.x + CELL * SIZE / 2) / CELL)
    const row = SIZE - 1 - Math.floor((local.y + CELL * SIZE / 2) / CELL)
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
    const x = -CELL * SIZE / 2 + col * CELL + CELL / 2
    const y = -CELL * SIZE / 2 + (SIZE - 1 - row) * CELL + CELL / 2
    return { x, y }
  }

  onBackToHall(): void {
    SessionStore.resetRoom()
    SessionStore.session?.leave()
    SessionStore.session = null
    liuzichongState.reset()
    director.loadScene('Lobby')
  }
}
