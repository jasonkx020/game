import { _decorator, Component, director, Label } from 'cc'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

@ccclass('HallScene')
export class HallScene extends Component {
  @property(Label)
  balanceLabel: Label | null = null

  @property(Label)
  userLabel: Label | null = null

  async start(): Promise<void> {
    if (!SessionStore.login) {
      director.loadScene('Launch')
      return
    }
    this.userLabel && (this.userLabel.string = `${SessionStore.login.nickname} (${SessionStore.login.role})`)
    try {
      const bal = await SessionStore.api.getRoomCardBalance()
      this.balanceLabel && (this.balanceLabel.string = `房卡: ${bal}`)
    } catch (e) {
      console.error('[Hall] balance', e)
    }
  }

  async onCreateDawuguiRoom(): Promise<void> {
    await this.createRoom('dawugui', 4)
  }

  async onCreateLiuzichongRoom(): Promise<void> {
    await this.createRoom('liuzichong', 2)
  }

  /** @deprecated 兼容旧按钮绑定 */
  async onQuickCreateRoom(): Promise<void> {
    await this.onCreateDawuguiRoom()
  }

  private async createRoom(gameId: string, playerCount: number): Promise<void> {
    try {
      SessionStore.resetRoom()
      SessionStore.room = await SessionStore.api.createRoom({ gameId, playerCount })
      director.loadScene('Room')
    } catch (e) {
      console.error('[Hall] create room failed', e)
    }
  }
}
