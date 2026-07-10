import { _decorator, Component, director, Label } from 'cc'
import { CompanionPanel } from '../../platform/companion/CompanionPanel'
import { GameShelf } from '../../platform/lobby/GameShelf'
import { GameHost } from '../../platform/host/GameHost'
import { SessionStore } from '../SessionStore'
import type { LobbyGameItem } from '../../platform/sdk/ApiClient'

const { ccclass, property } = _decorator

@ccclass('LobbyScene')
export class LobbyScene extends Component {
  @property(Label)
  balanceLabel: Label | null = null

  @property(Label)
  userLabel: Label | null = null

  @property(Label)
  statusLabel: Label | null = null

  @property(CompanionPanel)
  companionPanel: CompanionPanel | null = null

  @property(GameShelf)
  gameShelf: GameShelf | null = null

  private allGames: LobbyGameItem[] = []
  private loading = false
  private companionSessionId = 0

  async start(): Promise<void> {
    if (!SessionStore.login) {
      director.loadScene('Launch')
      return
    }
    this.userLabel && (this.userLabel.string = `${SessionStore.login.nickname} (${SessionStore.login.role})`)
    if (this.companionPanel) {
      await this.companionPanel.init(() => SessionStore.api.accessToken)
      this.companionSessionId = this.companionPanel.getSessionId()
    }
    this.gameShelf?.setHandler((gameId) => void this.onEnterGame(gameId))
    await this.refresh()
  }

  async refresh(): Promise<void> {
    try {
      const [bal, games, recs] = await Promise.all([
        SessionStore.api.getRoomCardBalance(),
        SessionStore.api.listLobbyGames(),
        SessionStore.api.listLobbyRecommendations(),
      ])
      this.balanceLabel && (this.balanceLabel.string = `房卡: ${bal}`)
      this.allGames = games
      this.gameShelf?.render(games, recs)
      this.setStatus('')
    } catch (e) {
      console.error('[Lobby] refresh', e)
      this.setStatus(`加载失败: ${String(e)}`)
    }
  }

  async onEnterGame(gameId: string): Promise<void> {
    if (this.loading) return
    const game = this.allGames.find((g) => g.gameId === gameId)
    if (!game || !game.visible) return

    this.loading = true
    this.setStatus(`准备 ${game.name}...`)
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
    } finally {
      this.loading = false
    }
  }

  private setStatus(text: string): void {
    if (this.statusLabel) this.statusLabel.string = text
  }
}
