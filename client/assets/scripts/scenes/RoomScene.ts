import { _decorator, Component, director, Label } from 'cc'
import { registerDawuguiPushHandlers } from '../../games/dawugui/DawuguiPushHandler'
import { registerLiuzichongPushHandlers, liuzichongState } from '../../games/liuzichong/LiuzichongPushHandler'
import { GameSession } from '../../platform/sdk/GameSession'
import { encodeProto, decodeProto } from '../../platform/sdk/protoHelpers'
import { PassReq, PassRsp, PlayCardsReq, PlayCardsRsp } from '../../platform/generated/pitaya/pitaya/dawugui'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

@ccclass('RoomScene')
export class RoomScene extends Component {
  @property(Label)
  logLabel: Label | null = null

  async start(): Promise<void> {
    if (!SessionStore.login || !SessionStore.room) {
      director.loadScene('Hall')
      return
    }
    const room = SessionStore.room
    const login = SessionStore.login
    const gameId = room.gameId || 'dawugui'

    SessionStore.session?.leave()
    const session = new GameSession({
      api: SessionStore.api,
      onLog: (line) => {
        SessionStore.appendLog(line)
        this.refreshLog()
      },
    })
    SessionStore.session = session

    if (gameId === 'liuzichong') {
      liuzichongState.reset()
      registerLiuzichongPushHandlers(session.router, {
        log: (line) => {
          SessionStore.appendLog(line)
          this.refreshLog()
        },
        userId: login.userId,
      })
    } else {
      registerDawuguiPushHandlers(session.router, (line) => {
        SessionStore.appendLog(line)
        this.refreshLog()
      })
    }

    try {
      await session.enterRoom({
        wsUrl: room.wsUrl,
        roomId: room.roomId,
        accessToken: login.accessToken,
      })
      SessionStore.appendLog(`[room] entered ${room.roomId} game=${gameId}`)
      this.refreshLog()
      if (gameId === 'liuzichong') {
        director.loadScene('Liuzichong')
      }
    } catch (e) {
      console.error('[Room] enter failed', e)
      SessionStore.appendLog(`[error] ${String(e)}`)
      this.refreshLog()
    }
  }

  onDestroy(): void {
    // Liuzichong 场景接管 session；打乌龟仍在 Room
    if (SessionStore.room?.gameId !== 'liuzichong') {
      SessionStore.session?.leave()
      SessionStore.session = null
    }
  }

  async onPassClick(): Promise<void> {
    const session = SessionStore.session
    if (!session || !SessionStore.room) return
    const body = encodeProto(PassReq, { roomId: SessionStore.room.roomId })
    const rsp = await session.pitaya.request('game.dawugui.pass', body)
    decodeProto(PassRsp, rsp)
    SessionStore.appendLog('[action] pass sent')
    this.refreshLog()
  }

  async onPlayClick(): Promise<void> {
    const session = SessionStore.session
    if (!session || !SessionStore.room) return
    const body = encodeProto(PlayCardsReq, { roomId: SessionStore.room.roomId, cards: [1] })
    const rsp = await session.pitaya.request('game.dawugui.playcards', body)
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
