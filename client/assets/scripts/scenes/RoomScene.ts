import { _decorator, Component, director, Label } from 'cc'
import { registerDawuguiPushHandlers } from '../../games/dawugui/DawuguiPushHandler'
import { GameSession } from '../../platform/sdk/GameSession'
import { encodeProto, decodeProto } from '../../platform/sdk/protoHelpers'
import { PassReq, PassRsp, PlayCardsReq, PlayCardsRsp } from '../../platform/generated/pitaya/pitaya/dawugui'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

@ccclass('RoomScene')
export class RoomScene extends Component {
  @property(Label)
  logLabel: Label | null = null

  private session: GameSession | null = null

  async start(): Promise<void> {
    if (!SessionStore.login || !SessionStore.room) {
      director.loadScene('Hall')
      return
    }
    const room = SessionStore.room
    const login = SessionStore.login

    this.session = new GameSession({
      api: SessionStore.api,
      onLog: (line) => {
        SessionStore.appendLog(line)
        this.refreshLog()
      },
    })

    registerDawuguiPushHandlers(this.session.router, (line) => {
      SessionStore.appendLog(line)
      this.refreshLog()
    })

    try {
      await this.session.enterRoom({
        wsUrl: room.wsUrl,
        roomId: room.roomId,
        accessToken: login.accessToken,
      })
      SessionStore.appendLog(`[room] entered ${room.roomId}`)
      this.refreshLog()
    } catch (e) {
      console.error('[Room] enter failed', e)
      SessionStore.appendLog(`[error] ${String(e)}`)
      this.refreshLog()
    }
  }

  onDestroy(): void {
    this.session?.leave()
  }

  async onPassClick(): Promise<void> {
    if (!this.session || !SessionStore.room) return
    const body = encodeProto(PassReq, { roomId: SessionStore.room.roomId })
    const rsp = await this.session.pitaya.request('game.dawugui.pass', body)
    decodeProto(PassRsp, rsp)
    SessionStore.appendLog('[action] pass sent')
    this.refreshLog()
  }

  async onPlayClick(): Promise<void> {
    if (!this.session || !SessionStore.room) return
    const body = encodeProto(PlayCardsReq, { roomId: SessionStore.room.roomId, cards: [1] })
    const rsp = await this.session.pitaya.request('game.dawugui.playcards', body)
    decodeProto(PlayCardsRsp, rsp)
    SessionStore.appendLog('[action] playcards sent')
    this.refreshLog()
  }

  private refreshLog(): void {
    if (this.logLabel) {
      this.logLabel.string = SessionStore.pushLogs.slice(-8).join('\n')
    }
  }
}
