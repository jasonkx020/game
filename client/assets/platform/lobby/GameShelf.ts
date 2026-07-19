import { _decorator, Component, Label, Node, UITransform } from 'cc'
import type { LobbyGameItem } from '../sdk/ApiClient'
import {
  LobbyTheme,
  clearChildren,
  createLabelNode,
  createUINode,
  ensureUILayer,
  formatCoin,
  gameIcon,
  paintRoundRect,
  place,
} from './LobbyUIKit'

const { ccclass, property } = _decorator

export type GameTapHandler = (gameId: string) => void

export type LobbyRecommendation = { gameId: string; name: string; reason?: string; tag?: string }

@ccclass('GameShelf')
export class GameShelf extends Component {
  @property(Label)
  titleLabel: Label | null = null

  @property(Label)
  recentLabel: Label | null = null

  @property(Node)
  gridRoot: Node | null = null

  @property(Node)
  listRoot: Node | null = null

  @property(Node)
  recommendRoot: Node | null = null

  @property(Node)
  recentRoot: Node | null = null

  @property
  columns = 4

  @property
  cardWidth = 150

  @property
  cardHeight = 100

  @property
  cardGapX = 12

  @property
  cardGapY = 12

  private onTap: GameTapHandler | null = null
  private coinByGame = new Map<string, number>()

  setHandler(handler: GameTapHandler): void {
    this.onTap = handler
  }

  setCoinBalances(coins: Record<string, number>): void {
    this.coinByGame.clear()
    for (const [k, v] of Object.entries(coins)) this.coinByGame.set(k, v)
  }

  render(games: LobbyGameItem[], recommendations: LobbyRecommendation[]): void {
    if (this.titleLabel) {
      this.titleLabel.string = '🎮 游戏架 · 置顶优先'
    }
    const visible = games.filter((g) => g.visible)
    const sorted = [...visible].sort((a, b) => {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
      return a.sortOrder - b.sortOrder
    })

    if (this.gridRoot) {
      this.renderGrid(sorted)
    } else if (this.listRoot) {
      this.renderListFallback(sorted)
    }

    this.renderRecommendations(recommendations, games)
    this.renderRecent(games)
  }

  private renderGrid(games: LobbyGameItem[]): void {
    const root = this.gridRoot!
    clearChildren(root)
    const cols = Math.max(1, this.columns)
    const totalW = cols * this.cardWidth + (cols - 1) * this.cardGapX
    const startX = -totalW / 2 + this.cardWidth / 2
    // 顶部对齐：第一行在容器上半
    const startY = this.cardHeight / 2 + 8

    games.forEach((game, index) => {
      const col = index % cols
      const row = Math.floor(index / cols)
      const x = startX + col * (this.cardWidth + this.cardGapX)
      const y = startY - row * (this.cardHeight + this.cardGapY)
      const card = this.buildGameCard(game, index)
      place(card, x, y)
      root.addChild(card)
    })
  }

  private buildGameCard(game: LobbyGameItem, tintIndex: number): Node {
    const w = this.cardWidth
    const h = this.cardHeight
    const card = createUINode(`game-${game.gameId}`)
    card.addComponent(UITransform).setContentSize(w, h)
    const tint = LobbyTheme.cardTints[tintIndex % LobbyTheme.cardTints.length]
    paintRoundRect(card, w, h, tint, 16)

    if (game.pinned) {
      const pin = createLabelNode('pin', '📌', 12, LobbyTheme.gold, 22, 18)
      place(pin, w / 2 - 14, h / 2 - 12)
      card.addChild(pin)
    }

    const icon = createLabelNode('icon', gameIcon(game.gameId), 28, LobbyTheme.text, w - 8, 34)
    icon.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(icon, 0, 18)
    card.addChild(icon)

    const name = createLabelNode('name', game.name, 12, LobbyTheme.text, w - 10, 20)
    name.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(name, 0, -10)
    card.addChild(name)

    const coinVal = this.coinByGame.get(game.gameId)
    const coinText =
      coinVal !== undefined
        ? formatCoin(coinVal)
        : `${game.minPlayers}-${game.maxPlayers}人`

    const coinBg = createUINode('coinBg')
    coinBg.addComponent(UITransform).setContentSize(56, 18)
    paintRoundRect(coinBg, 56, 18, LobbyTheme.roomCardBg, 9)
    place(coinBg, 0, -34)
    card.addChild(coinBg)
    const coin = createLabelNode('coin', coinText, 10, LobbyTheme.gold, 52, 16)
    coin.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    place(coin, 0, 0)
    coinBg.addChild(coin)

    card.on(Node.EventType.TOUCH_END, () => this.onTap?.(game.gameId))
    ensureUILayer(card)
    return card
  }

  private renderListFallback(games: LobbyGameItem[]): void {
    const root = this.listRoot!
    clearChildren(root)
    games.forEach((game, index) => {
      const row = createUINode(`game-${game.gameId}`)
      row.addComponent(UITransform).setContentSize(600, 44)
      place(row, 0, -index * 48)
      const label = row.addComponent(Label)
      const pin = game.pinned ? '★ ' : ''
      label.string = `${pin}${gameIcon(game.gameId)} ${game.name}`
      label.fontSize = 18
      label.color = LobbyTheme.text
      row.on(Node.EventType.TOUCH_END, () => this.onTap?.(game.gameId))
      ensureUILayer(row)
      root.addChild(row)
    })
  }

  /** 横向推荐条（对齐 HTML recommend-scroll） */
  private renderRecommendations(recs: LobbyRecommendation[], games: LobbyGameItem[]): void {
    if (!this.recommendRoot) {
      if (this.titleLabel && recs[0]) {
        this.titleLabel.string = `游戏架 · 推荐 ${recs[0].name}`
      }
      return
    }
    clearChildren(this.recommendRoot)
    const byId = new Map(games.map((g) => [g.gameId, g]))
    const itemW = 280
    const itemH = 80
    const gap = 14
    const list = recs.slice(0, 6)
    const totalW = list.length * itemW + Math.max(0, list.length - 1) * gap
    let x = -totalW / 2 + itemW / 2

    list.forEach((r) => {
      const item = createUINode(`rec-${r.gameId}`)
      item.addComponent(UITransform).setContentSize(itemW, itemH)
      paintRoundRect(item, itemW, itemH, LobbyTheme.cardBg, 18)
      place(item, x, 0)

      const iconBg = createUINode('iconBg')
      iconBg.addComponent(UITransform).setContentSize(56, 56)
      paintRoundRect(iconBg, 56, 56, LobbyTheme.roomCardBg, 16)
      place(iconBg, -itemW / 2 + 40, 0)
      item.addChild(iconBg)
      const icon = createLabelNode('icon', gameIcon(r.gameId), 28, LobbyTheme.gold, 48, 48)
      icon.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
      iconBg.addChild(icon)

      const name = createLabelNode('name', r.name, 15, LobbyTheme.text, 140, 24)
      name.getComponent(Label)!.isBold = true
      place(name, 20, 12)
      item.addChild(name)

      const desc = createLabelNode(
        'desc',
        r.reason || byId.get(r.gameId)?.description || '推荐游玩',
        11,
        LobbyTheme.textMuted,
        150,
        20,
      )
      place(desc, 20, -12)
      item.addChild(desc)

      const tagBg = createUINode('tagBg')
      tagBg.addComponent(UITransform).setContentSize(44, 20)
      paintRoundRect(tagBg, 44, 20, LobbyTheme.roomCardBg, 10)
      place(tagBg, itemW / 2 - 36, 0)
      item.addChild(tagBg)
      const tag = createLabelNode('tag', r.tag || '荐', 10, LobbyTheme.gold, 40, 18)
      tag.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
      tagBg.addChild(tag)

      item.on(Node.EventType.TOUCH_END, () => this.onTap?.(r.gameId))
      ensureUILayer(item)
      this.recommendRoot!.addChild(item)
      x += itemW + gap
    })
  }

  /** 最近玩过标签（对齐 HTML recent-tags） */
  private renderRecent(games: LobbyGameItem[]): void {
    const recent = games
      .filter((g) => g.lastPlayedAt)
      .sort((a, b) => String(b.lastPlayedAt).localeCompare(String(a.lastPlayedAt)))
      .slice(0, 3)

    if (!this.recentRoot) {
      if (this.recentLabel) {
        this.recentLabel.string =
          recent.length === 0 ? '最近玩过：暂无' : `最近玩过：${recent.map((g) => g.name).join(' · ')}`
      }
      return
    }

    clearChildren(this.recentRoot)
    if (recent.length === 0) {
      const empty = createLabelNode('empty', '暂无', 13, LobbyTheme.textFaint, 80, 28)
      this.recentRoot.addChild(empty)
      return
    }

    let x = -this.recentRoot.getComponent(UITransform)!.contentSize.width / 2 + 8
    recent.forEach((g) => {
      const text = `${gameIcon(g.gameId)} ${g.name}`
      const w = Math.min(160, 28 + g.name.length * 14)
      const h = 34
      const tag = createUINode(`recent-${g.gameId}`)
      tag.addComponent(UITransform).setContentSize(w, h)
      paintRoundRect(tag, w, h, LobbyTheme.cardBg, 17)
      place(tag, x + w / 2, 0)
      const label = createLabelNode('t', text, 13, LobbyTheme.text, w - 8, 28)
      label.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
      tag.addChild(label)
      tag.on(Node.EventType.TOUCH_END, () => this.onTap?.(g.gameId))
      ensureUILayer(tag)
      this.recentRoot!.addChild(tag)
      x += w + 10
    })
  }
}
