import { _decorator, Component, Label, Node, UITransform, Vec3 } from 'cc'
import type { LobbyGameItem } from '../sdk/ApiClient'

const { ccclass, property } = _decorator

export type GameTapHandler = (gameId: string) => void

@ccclass('GameShelf')
export class GameShelf extends Component {
  @property(Label)
  titleLabel: Label | null = null

  @property(Node)
  listRoot: Node | null = null

  private onTap: GameTapHandler | null = null

  setHandler(handler: GameTapHandler): void {
    this.onTap = handler
  }

  render(games: LobbyGameItem[], recommendations: Array<{ gameId: string; name: string; reason?: string }>): void {
    if (this.titleLabel) {
      const rec = recommendations[0]
      this.titleLabel.string = rec ? `推荐：${rec.name}` : '全部游戏'
    }
    if (!this.listRoot) return
    this.listRoot.removeAllChildren()
    const visible = games.filter((g) => g.visible)
    visible.forEach((game, index) => {
      const row = new Node(`game-${game.gameId}`)
      row.addComponent(UITransform).setContentSize(600, 44)
      row.setPosition(new Vec3(0, -index * 48, 0))
      const label = row.addComponent(Label)
      const pin = game.pinned ? '★ ' : ''
      label.string = `${pin}${game.name} (${game.minPlayers}-${game.maxPlayers}人)`
      label.fontSize = 20
      row.on(Node.EventType.TOUCH_END, () => this.onTap?.(game.gameId))
      this.listRoot.addChild(row)
    })
  }
}
