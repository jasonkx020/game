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

  async onQuickCreateRoom(): Promise<void> {
    try {
      SessionStore.room = await SessionStore.api.createRoom({
        gameId: 'dawugui',
        playerCount: 4,
      })
      director.loadScene('Room')
    } catch (e) {
      console.error('[Hall] create room failed', e)
    }
  }
}
