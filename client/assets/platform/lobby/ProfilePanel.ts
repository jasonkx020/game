import { Label, Node, UITransform } from 'cc'
import {
  LobbyTheme,
  UI_DESIGN_HEIGHT,
  UI_DESIGN_WIDTH,
  createLabelNode,
  createUINode,
  ensureUILayer,
  paintRoundRect,
  paintSkyBackground,
  place,
} from './LobbyUIKit'
import type { LobbyGameItem, MatchSummary, RechargeOrder, UserProfile } from '../sdk/ApiClient'
import { SessionStore } from '../../scripts/SessionStore'

export type ProfilePanelHandlers = {
  onBack: () => void
  onLogout: () => void
}

const W = UI_DESIGN_WIDTH
const H = UI_DESIGN_HEIGHT
const COL_W = W - 64

/**
 * 个人页面板（720×1280 竖屏单列）。
 */
export class ProfilePanel {
  root: Node
  private nicknameLabel: Label
  private phoneLabel: Label
  private avatarLabel: Label
  private assetsLabel: Label
  private historyLabel: Label
  private prefsLabel: Label
  private settingsLabel: Label
  private matchesLabel: Label
  private statusLabel: Label
  private nicknameDraft = ''
  private games: LobbyGameItem[] = []
  private settings: Record<string, unknown> = {}

  constructor(parent: Node, handlers: ProfilePanelHandlers) {
    const old = parent.getChildByName('ProfileOverlay')
    if (old) old.destroy()

    const root = createUINode('ProfileOverlay')
    root.addComponent(UITransform).setContentSize(W, H)
    place(root, 0, 0)
    parent.addChild(root)
    paintSkyBackground(root, W, H)
    this.root = root
    root.active = false

    const title = createLabelNode('Title', '👤 我的', 22, LobbyTheme.brand, 200, 36)
    title.getComponent(Label)!.isBold = true
    place(title, -W / 2 + 120, H / 2 - 56)
    root.addChild(title)

    const back = createLabelNode('Back', '‹ 返回大厅', 16, LobbyTheme.gold, 140, 32)
    place(back, W / 2 - 100, H / 2 - 56)
    back.on(Node.EventType.TOUCH_END, handlers.onBack)
    root.addChild(back)

    let y = H / 2 - 120
    this.nicknameLabel = addBlock(root, 'Nickname', 0, y, COL_W, 40, '昵称：—').getComponent(Label)!
    y -= 52
    this.phoneLabel = addBlock(root, 'Phone', 0, y, COL_W, 36, '手机：—').getComponent(Label)!
    y -= 48
    this.avatarLabel = addBlock(root, 'Avatar', 0, y, COL_W, 36, '头像：—').getComponent(Label)!
    y -= 48

    const saveNick = createLabelNode('SaveNick', '[ 保存昵称+1 ]', 14, LobbyTheme.gold, 160, 30)
    place(saveNick, 0, y)
    saveNick.on(Node.EventType.TOUCH_END, () => void this.saveNicknameBump())
    root.addChild(saveNick)
    y -= 56

    this.assetsLabel = addBlock(root, 'Assets', 0, y, COL_W, 80, '资产：—').getComponent(Label)!
    this.assetsLabel.overflow = Label.Overflow.SHRINK
    this.assetsLabel.enableWrapText = true
    this.assetsLabel.verticalAlign = Label.VerticalAlign.TOP
    y -= 96

    this.historyLabel = addBlock(root, 'History', 0, y, COL_W, 70, '充值：—').getComponent(Label)!
    this.historyLabel.overflow = Label.Overflow.SHRINK
    this.historyLabel.enableWrapText = true
    this.historyLabel.verticalAlign = Label.VerticalAlign.TOP
    y -= 86

    this.prefsLabel = addBlock(root, 'Prefs', 0, y, COL_W, 80, '偏好：—').getComponent(Label)!
    this.prefsLabel.overflow = Label.Overflow.SHRINK
    this.prefsLabel.enableWrapText = true
    this.prefsLabel.verticalAlign = Label.VerticalAlign.TOP
    y -= 90

    const toggleVis = createLabelNode('ToggleVis', '[ 切换首游戏可见 ]', 13, LobbyTheme.goldDim, 180, 28)
    place(toggleVis, -120, y)
    toggleVis.on(Node.EventType.TOUCH_END, () => void this.togglePrefVisible())
    root.addChild(toggleVis)
    const togglePin = createLabelNode('TogglePin', '[ 切换首游戏置顶 ]', 13, LobbyTheme.goldDim, 180, 28)
    place(togglePin, 120, y)
    togglePin.on(Node.EventType.TOUCH_END, () => void this.togglePrefPinned())
    root.addChild(togglePin)
    y -= 50

    this.settingsLabel = addBlock(root, 'Settings', 0, y, COL_W, 60, '设置：—').getComponent(Label)!
    this.settingsLabel.enableWrapText = true
    this.settingsLabel.verticalAlign = Label.VerticalAlign.TOP
    y -= 72

    const soundBtn = createLabelNode('Sound', '[ 音效开关 ]', 13, LobbyTheme.goldDim, 120, 28)
    place(soundBtn, -100, y)
    soundBtn.on(Node.EventType.TOUCH_END, () => void this.toggleSetting('sound_enabled'))
    root.addChild(soundBtn)
    const companionBtn = createLabelNode('CompanionSet', '[ 伴侣开关 ]', 13, LobbyTheme.goldDim, 120, 28)
    place(companionBtn, 100, y)
    companionBtn.on(Node.EventType.TOUCH_END, () => void this.toggleSetting('companion_enabled'))
    root.addChild(companionBtn)
    y -= 56

    this.matchesLabel = addBlock(root, 'Matches', 0, y, COL_W, 100, '战绩：—').getComponent(Label)!
    this.matchesLabel.enableWrapText = true
    this.matchesLabel.verticalAlign = Label.VerticalAlign.TOP
    this.matchesLabel.overflow = Label.Overflow.SHRINK

    this.statusLabel = createLabelNode('Status', '', 13, LobbyTheme.textMuted, COL_W, 24).getComponent(Label)!
    place(this.statusLabel.node, 0, -H / 2 + 90)
    this.statusLabel.horizontalAlign = Label.HorizontalAlign.CENTER
    root.addChild(this.statusLabel.node)

    const logout = createLabelNode('Logout', '[ 退出登录 ]', 15, LobbyTheme.danger, 140, 32)
    place(logout, 0, -H / 2 + 50)
    logout.on(Node.EventType.TOUCH_END, handlers.onLogout)
    root.addChild(logout)

    ensureUILayer(root)
  }

  show(): void {
    this.root.active = true
    void this.refresh()
  }

  hide(): void {
    this.root.active = false
  }

  async refresh(): Promise<void> {
    this.statusLabel.string = '加载中...'
    try {
      const [profile, roomCard, dawuguiCoin, liuzichongCoin, history, games, settings, matches] =
        await Promise.all([
          SessionStore.api.getProfile(),
          SessionStore.api.getRoomCardBalance(),
          SessionStore.api.getGameCoinBalance('dawugui').catch(() => 0),
          SessionStore.api.getGameCoinBalance('liuzichong').catch(() => 0),
          SessionStore.api.getRechargeHistory().catch(() => [] as RechargeOrder[]),
          SessionStore.api.listLobbyGames(),
          SessionStore.api.getUserSettings(),
          SessionStore.api.listMyMatches({ page: 1, pageSize: 8 }),
        ])
      SessionStore.profile = profile
      if (SessionStore.login) SessionStore.login.nickname = profile.nickname
      this.nicknameDraft = profile.nickname
      this.games = games
      this.settings = settings
      this.applyProfile(profile)
      this.assetsLabel.string = `资产：\n房卡 ${roomCard}  ·  打乌龟 ${dawuguiCoin}  ·  六子冲 ${liuzichongCoin}`
      this.historyLabel.string = formatHistory(history)
      this.prefsLabel.string = formatPrefs(games)
      this.settingsLabel.string = formatSettings(settings)
      this.matchesLabel.string = formatMatches(matches.items)
      this.statusLabel.string = ''
    } catch (e) {
      console.error('[ProfilePanel]', e)
      this.statusLabel.string = `加载失败: ${String(e)}`
    }
  }

  private applyProfile(profile: UserProfile): void {
    this.nicknameLabel.string = `昵称：${profile.nickname}`
    this.phoneLabel.string = `手机：${profile.phoneMasked || '****'}`
    this.avatarLabel.string = `头像：${profile.avatarUrl || '（默认）'}`
  }

  private async saveNicknameBump(): Promise<void> {
    const base = (this.nicknameDraft || '玩家').replace(/\+\d+$/, '')
    const next = `${base}+${Date.now() % 100}`
    try {
      const profile = await SessionStore.api.updateProfile({ nickname: next })
      SessionStore.profile = profile
      if (SessionStore.login) SessionStore.login.nickname = profile.nickname
      this.nicknameDraft = profile.nickname
      this.applyProfile(profile)
      this.statusLabel.string = '昵称已保存'
    } catch (e) {
      this.statusLabel.string = `保存失败: ${String(e)}`
    }
  }

  private async togglePrefVisible(): Promise<void> {
    const game = this.games[0]
    if (!game) return
    try {
      this.games = await SessionStore.api.updateLobbyGames([
        { gameId: game.gameId, visible: !game.visible },
      ])
      this.prefsLabel.string = formatPrefs(this.games)
      this.statusLabel.string = `${game.name} 可见=${!game.visible}`
    } catch (e) {
      this.statusLabel.string = `偏好失败: ${String(e)}`
    }
  }

  private async togglePrefPinned(): Promise<void> {
    const game = this.games[0]
    if (!game) return
    try {
      this.games = await SessionStore.api.updateLobbyGames([
        { gameId: game.gameId, pinned: !game.pinned },
      ])
      this.prefsLabel.string = formatPrefs(this.games)
      this.statusLabel.string = `${game.name} 置顶=${!game.pinned}`
    } catch (e) {
      this.statusLabel.string = `偏好失败: ${String(e)}`
    }
  }

  private async toggleSetting(key: string): Promise<void> {
    const cur = Boolean(this.settings[key])
    try {
      this.settings = await SessionStore.api.updateUserSettings({ [key]: !cur })
      this.settingsLabel.string = formatSettings(this.settings)
      this.statusLabel.string = `${key} = ${!cur}`
    } catch (e) {
      this.statusLabel.string = `设置失败: ${String(e)}`
    }
  }
}

function addBlock(parent: Node, name: string, x: number, y: number, w: number, h: number, text: string): Node {
  const node = createUINode(name)
  node.addComponent(UITransform).setContentSize(w, h)
  paintRoundRect(node, w, h, LobbyTheme.surface, 22)
  place(node, x, y)
  const labelNode = createLabelNode('L', text, 14, LobbyTheme.text, w - 20, h - 12)
  labelNode.getComponent(Label)!.verticalAlign = Label.VerticalAlign.CENTER
  parent.addChild(node)
  node.addChild(labelNode)
  return labelNode
}

function formatHistory(orders: RechargeOrder[]): string {
  if (!orders.length) return '充值记录：暂无'
  return `充值记录：\n${orders
    .slice(0, 4)
    .map((o) => `${o.productId} +${o.cards} ¥${o.amountCny}`)
    .join('\n')}`
}

function formatPrefs(games: LobbyGameItem[]): string {
  if (!games.length) return '游戏偏好：暂无'
  return `游戏偏好：\n${games
    .map((g) => `${g.name}(${g.visible ? '显' : '隐'}${g.pinned ? '/顶' : ''})`)
    .join(' · ')}`
}

function formatSettings(settings: Record<string, unknown>): string {
  const sound = settings.sound_enabled === undefined ? true : Boolean(settings.sound_enabled)
  const companion = settings.companion_enabled === undefined ? true : Boolean(settings.companion_enabled)
  return `设置：音效=${sound ? '开' : '关'} · 伴侣=${companion ? '开' : '关'}`
}

function formatMatches(matches: MatchSummary[]): string {
  if (!matches.length) return '战绩：暂无'
  return `战绩：\n${matches
    .slice(0, 5)
    .map((m) => `${m.gameId} R${m.roundNo} ${m.isWinner ? '胜' : m.status} 分${m.myRuleScore}`)
    .join('\n')}`
}
