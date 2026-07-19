import { _decorator, Component, director, EditBox, Label, Node } from 'cc'
import { CompanionPanel } from '../../platform/companion/CompanionPanel'
import { GameShelf } from '../../platform/lobby/GameShelf'
import { buildLobbyUI } from '../../platform/lobby/LobbyAutoBuilder'
import { ProfilePanel } from '../../platform/lobby/ProfilePanel'
import { GameHost } from '../../platform/host/GameHost'
import { LobbyTheme } from '../../platform/lobby/LobbyUIKit'
import { resolveOrCreateUICanvas } from '../../platform/lobby/UICanvasHost'
import { SessionStore } from '../SessionStore'
import type { LobbyGameItem } from '../../platform/sdk/ApiClient'

const { ccclass, property } = _decorator

@ccclass('LobbyScene')
export class LobbyScene extends Component {
  /** 默认 true：运行时用代码搭完整大厅，无需在编辑器拖引用 */
  @property
  autoBuildUI = true

  @property(Label)
  balanceLabel: Label | null = null

  @property(Label)
  userLabel: Label | null = null

  @property(Label)
  avatarHintLabel: Label | null = null

  @property(Label)
  onlineLabel: Label | null = null

  @property(Label)
  statusLabel: Label | null = null

  @property(Label)
  toastLabel: Label | null = null

  @property(Label)
  toastSubLabel: Label | null = null

  @property(Node)
  toastRoot: Node | null = null

  @property(Node)
  avatarMenuRoot: Node | null = null

  @property(Node)
  companionPanelRoot: Node | null = null

  @property(Label)
  companionBadgeLabel: Label | null = null

  @property(Label)
  clubLabel: Label | null = null

  @property(CompanionPanel)
  companionPanel: CompanionPanel | null = null

  @property(GameShelf)
  gameShelf: GameShelf | null = null

  private allGames: LobbyGameItem[] = []
  private loading = false
  private companionSessionId = 0
  private menuOpen = false
  private companionOpen = false
  private toastTimer: ReturnType<typeof setTimeout> | null = null
  private built = false
  private profilePanel: ProfilePanel | null = null
  private matchPanelRoot: Node | null = null
  private matchRoomInput: EditBox | null = null
  private matchGame: LobbyGameItem | null = null
  private matchTitleLabel: Label | null = null

  onLoad(): void {
    if (this.autoBuildUI) {
      this.mountAutoUI()
    }
  }

  async start(): Promise<void> {
    if (!SessionStore.login) {
      director.loadScene('Launch')
      return
    }
    if (this.autoBuildUI && !this.built) {
      this.mountAutoUI()
    }

    this.hideToast()
    this.closeAvatarMenu()
    if (this.companionPanelRoot) this.companionPanelRoot.active = false
    if (this.onlineLabel) {
      this.onlineLabel.string = '● 在线'
      this.onlineLabel.color = LobbyTheme.online
    }
    if (this.clubLabel) this.clubLabel.string = '星辰棋牌社'

    if (this.companionPanel) {
      try {
        await this.companionPanel.init(() => SessionStore.api.accessToken)
        this.companionSessionId = this.companionPanel.getSessionId()
        if (this.companionBadgeLabel) {
          this.companionBadgeLabel.string = '●'
          this.companionBadgeLabel.color = LobbyTheme.online
        }
      } catch (e) {
        console.warn('[Lobby] companion init', e)
        if (this.companionBadgeLabel) {
          this.companionBadgeLabel.string = '○'
          this.companionBadgeLabel.color = LobbyTheme.textFaint
        }
      }
    }
    this.gameShelf?.setHandler((gameId) => void this.onEnterGame(gameId))
    await this.refresh()
  }

  private mountAutoUI(): void {
    const host = this.resolveUIHost()
    const ui = buildLobbyUI(host, {
      onUserInfo: () => this.onUserInfoClick(),
      onRoomCard: () => this.onRoomCardClick(),
      onCompanion: () => this.onCompanionEntryClick(),
      onProfile: () => this.onOpenProfileClick(),
      onLogout: () => this.onLogoutClick(),
      onEditGames: () => this.onEditGamesClick(),
      onClub: () => this.onClubFooterClick(),
      onMatchVsBot: () => void this.launchMatch({ fillBots: true }),
      onMatchCreatePvp: () => void this.launchMatch({ fillBots: false }),
      onMatchJoin: () => void this.launchMatchJoin(),
      onMatchClose: () => this.closeMatchPanel(),
    })
    this.userLabel = ui.userLabel
    this.balanceLabel = ui.balanceLabel
    this.onlineLabel = ui.onlineLabel
    this.statusLabel = ui.statusLabel
    this.toastRoot = ui.toastRoot
    this.toastLabel = ui.toastLabel
    this.toastSubLabel = ui.toastSubLabel
    this.avatarMenuRoot = ui.avatarMenuRoot
    this.companionPanelRoot = ui.companionPanelRoot
    this.companionBadgeLabel = ui.companionBadgeLabel
    this.clubLabel = ui.clubLabel
    this.gameShelf = ui.gameShelf
    this.companionPanel = ui.companionPanel
    this.matchPanelRoot = ui.matchPanelRoot
    this.matchRoomInput = ui.matchRoomInput
    this.matchTitleLabel = ui.matchTitleLabel
    this.profilePanel = new ProfilePanel(ui.root, {
      onBack: () => {
        this.profilePanel?.hide()
        void this.refresh()
      },
      onLogout: () => this.onLogoutClick(),
    })
    this.built = true
    this.hideLegacySiblings(host, ui.root)
  }

  private resolveUIHost(): Node {
    // Lobby.scene 可能没有 Canvas；缺则运行时创建，否则代码 UI 不可见
    return resolveOrCreateUICanvas(this.node)
  }

  private hideLegacySiblings(host: Node, keep: Node): void {
    for (const child of [...host.children]) {
      if (child === keep) continue
      const n = child.name
      if (n === 'Camera' || n === 'Main Camera' || n.includes('Camera')) continue
      if (child.getComponent('cc.Camera')) continue
      child.active = false
    }
  }

  async refresh(): Promise<void> {
    try {
      const [profile, bal, games, recs] = await Promise.all([
        SessionStore.api.getProfile(),
        SessionStore.api.getRoomCardBalance(),
        SessionStore.api.listLobbyGames(),
        SessionStore.api.listLobbyRecommendations(),
      ])
      SessionStore.profile = profile
      if (SessionStore.login) {
        SessionStore.login.nickname = profile.nickname
      }
      this.userLabel && (this.userLabel.string = profile.nickname)
      this.avatarHintLabel && (this.avatarHintLabel.string = profile.avatarUrl ? '头像' : '默认')
      this.balanceLabel && (this.balanceLabel.string = String(bal))

      const visibleIds = games.filter((g) => g.visible).map((g) => g.gameId)
      const coinEntries = await Promise.all(
        visibleIds.map(async (id) => {
          try {
            const n = await SessionStore.api.getGameCoinBalance(id)
            return [id, n] as const
          } catch {
            return [id, 0] as const
          }
        }),
      )
      const coins: Record<string, number> = {}
      for (const [id, n] of coinEntries) coins[id] = n
      this.gameShelf?.setCoinBalances(coins)

      this.allGames = games
      this.gameShelf?.render(games, recs)
      this.setStatus('')
    } catch (e) {
      console.error('[Lobby] refresh', e)
      this.setStatus(`加载失败: ${String(e)}`)
      this.showToast('加载失败', String(e))
    }
  }

  onUserInfoClick(): void {
    if (this.menuOpen) this.closeAvatarMenu()
    else this.openAvatarMenu()
  }

  onOpenProfileClick(): void {
    this.closeAvatarMenu()
    if (this.profilePanel) {
      this.profilePanel.show()
      return
    }
    director.loadScene('Profile')
  }

  onLogoutClick(): void {
    this.closeAvatarMenu()
    SessionStore.logout()
    director.loadScene('Launch')
  }

  onRoomCardClick(): void {
    const bal = this.balanceLabel?.string || '0'
    this.showToast(`房卡余额 ${bal}`, '明细请在「我的」中查看')
  }

  onCompanionEntryClick(): void {
    this.companionOpen = !this.companionOpen
    if (this.companionPanelRoot) {
      this.companionPanelRoot.active = this.companionOpen
    }
    this.showToast(this.companionOpen ? '伴侣已打开' : '伴侣已收起', '小龟陪你玩')
  }

  onEditGamesClick(): void {
    this.showToast('游戏架编辑', '打开个人页调整偏好')
    this.onOpenProfileClick()
  }

  onClubFooterClick(): void {
    this.showToast('俱乐部', '功能开发中')
  }

  async onEnterGame(gameId: string): Promise<void> {
    if (this.loading) return
    const game = this.allGames.find((g) => g.gameId === gameId)
    if (!game || !game.visible) return

    if (gameId === 'liuzichong') {
      this.matchGame = game
      if (this.matchTitleLabel) this.matchTitleLabel.string = game.name
      if (this.matchPanelRoot) this.matchPanelRoot.active = true
      return
    }

    this.loading = true
    this.setStatus(`准备 ${game.name}...`)
    this.showToast(`进入 ${game.name}`, '正在开房...')
    try {
      await GameHost.launch({
        mode: 'lobby',
        gameId,
        game,
        companionSessionId: this.companionSessionId || SessionStore.companionSessionId || undefined,
      })
    } catch (e) {
      console.error('[Lobby] enter game', e)
      this.setStatus(`进入失败: ${String(e)}`)
      this.showToast('进入失败', String(e))
    } finally {
      this.loading = false
    }
  }

  private closeMatchPanel(): void {
    if (this.matchPanelRoot) this.matchPanelRoot.active = false
    this.matchGame = null
  }

  private async launchMatch(opts: { fillBots: boolean }): Promise<void> {
    const game = this.matchGame
    if (!game || this.loading) return
    this.loading = true
    this.closeMatchPanel()
    this.setStatus(`准备 ${game.name}...`)
    this.showToast(
      opts.fillBots ? '人机对战' : '创建房间',
      opts.fillBots ? '正在开局…' : '创建后把房号发给好友',
    )
    try {
      await GameHost.launch({
        mode: 'lobby',
        gameId: game.gameId,
        game,
        fillBots: opts.fillBots,
        companionSessionId: this.companionSessionId || SessionStore.companionSessionId || undefined,
      })
    } catch (e) {
      console.error('[Lobby] match', e)
      this.setStatus(`进入失败: ${String(e)}`)
      this.showToast('进入失败', String(e))
    } finally {
      this.loading = false
    }
  }

  private async launchMatchJoin(): Promise<void> {
    const game = this.matchGame
    const roomId = (this.matchRoomInput?.string || '').trim()
    if (!game || this.loading) return
    if (!roomId) {
      this.showToast('请输入房间 ID', '')
      return
    }
    this.loading = true
    this.closeMatchPanel()
    this.setStatus('加入房间...')
    this.showToast('加入房间', roomId.slice(0, 8) + '…')
    try {
      await GameHost.launch({
        mode: 'lobby',
        gameId: game.gameId,
        game,
        joinRoomId: roomId,
        companionSessionId: this.companionSessionId || SessionStore.companionSessionId || undefined,
      })
    } catch (e) {
      console.error('[Lobby] join', e)
      this.setStatus(`加入失败: ${String(e)}`)
      this.showToast('加入失败', String(e))
    } finally {
      this.loading = false
    }
  }

  private openAvatarMenu(): void {
    this.menuOpen = true
    if (this.avatarMenuRoot) this.avatarMenuRoot.active = true
  }

  private closeAvatarMenu(): void {
    this.menuOpen = false
    if (this.avatarMenuRoot) this.avatarMenuRoot.active = false
  }

  private showToast(text: string, sub = ''): void {
    if (this.toastLabel) this.toastLabel.string = text
    if (this.toastSubLabel) this.toastSubLabel.string = sub
    if (this.toastRoot) this.toastRoot.active = true
    if (this.toastTimer) clearTimeout(this.toastTimer)
    this.toastTimer = setTimeout(() => this.hideToast(), 1600)
  }

  private hideToast(): void {
    if (this.toastRoot) this.toastRoot.active = false
  }

  private setStatus(text: string): void {
    if (this.statusLabel) this.statusLabel.string = text
  }

  onDestroy(): void {
    if (this.toastTimer) clearTimeout(this.toastTimer)
  }
}
