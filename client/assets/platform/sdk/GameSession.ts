import { ApiClient } from './ApiClient'
import { attachEventTracker, EventTracker } from './EventTracker'
import { PitayaClient } from './PitayaClient'
import { encodeProto, decodeProto } from './protoHelpers'
import { PushRouter } from './PushRouter'

import { EntryReq, EntryRsp, BindReq, BindRsp } from '../generated/pitaya/pitaya/connector'
import { JoinReq, JoinRsp, ReadyReq, ReadyRsp, SyncReq, SyncRsp } from '../generated/pitaya/pitaya/room'

const DEVICE_ID_KEY = 'games.device_id'

export interface EnterRoomParams {
  wsUrl: string
  roomId: string
  accessToken: string
  /** Optional WGS84 coords; omit to skip GPS risk check */
  lat?: number
  lng?: number
}

export interface GameSessionOptions {
  api?: ApiClient
  onLog?: (line: string) => void
}

export interface EnterRoomResult {
  seat: number
  gameId: string
}

function ensureDeviceId(): string {
  try {
    const existing = localStorage.getItem(DEVICE_ID_KEY)
    if (existing && existing.length >= 8) {
      return existing
    }
  } catch {
    /* ignore */
  }
  const id =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `dev-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
  try {
    localStorage.setItem(DEVICE_ID_KEY, id)
  } catch {
    /* ignore */
  }
  return id
}

export class GameSession {
  readonly pitaya = new PitayaClient()
  readonly tracker = new EventTracker()
  readonly router = new PushRouter()
  private api: ApiClient
  private log: (line: string) => void
  roomId = ''
  mySeat = -1
  deviceId = ensureDeviceId()

  constructor(opts: GameSessionOptions = {}) {
    this.api = opts.api ?? new ApiClient()
    this.log = opts.onLog ?? ((s) => console.log(s))
  }

  get apiClient(): ApiClient {
    return this.api
  }

  async enterRoom(params: EnterRoomParams): Promise<EnterRoomResult> {
    this.roomId = params.roomId
    this.tracker.reset()
    attachEventTracker(this.pitaya, this.tracker)
    this.tracker.setSyncFn(async (roomId, roundId, since) => {
      this.log(`[sync] room=${roomId} round=${roundId} since=${since}`)
      const req: SyncReq = { roomId, roundId, sinceActionSeq: since }
      const body = encodeProto(SyncReq, req)
      const rspBytes = await this.pitaya.request('game.room.sync', body)
      const rsp = decodeProto(SyncRsp, rspBytes)
      this.log(`[sync] latest=${rsp.latestActionSeq} pushes=${rsp.pushes.length}`)
    })

    this.router.bindToClient(this.pitaya)

    await this.pitaya.connect(params.wsUrl)
    this.log('[pitaya] connected')

    const entryBytes = encodeProto(EntryReq, { clientVersion: '0.1.0', platform: 'cocos' })
    const entryRspBytes = await this.pitaya.request('game.connector.entry', entryBytes)
    const entryRsp = decodeProto(EntryRsp, entryRspBytes)
    this.log(`[pitaya] entry ok server=${entryRsp.serverVersion}`)

    const bindBytes = encodeProto(BindReq, {
      accessToken: params.accessToken,
      deviceId: this.deviceId,
    })
    const bindRspBytes = await this.pitaya.request('game.connector.bind', bindBytes)
    const bindRsp = decodeProto(BindRsp, bindRspBytes)
    this.log(`[pitaya] bind uid=${bindRsp.userId}`)

    const joinReq: JoinReq = {
      roomId: params.roomId,
      asSpectator: false,
    }
    if (typeof params.lat === 'number' && typeof params.lng === 'number') {
      joinReq.lat = params.lat
      joinReq.lng = params.lng
    }
    const joinBytes = encodeProto(JoinReq, joinReq)
    const joinRspBytes = await this.pitaya.request('game.room.join', joinBytes)
    const joinRsp = decodeProto(JoinRsp, joinRspBytes)
    this.mySeat = joinRsp.seat ?? 0
    this.log(`[pitaya] join game=${joinRsp.gameId} seat=${this.mySeat}`)

    const readyBytes = encodeProto(ReadyReq, { roomId: params.roomId })
    await this.pitaya.request('game.room.ready', readyBytes)
    decodeProto(ReadyRsp, new Uint8Array())
    this.log('[pitaya] ready sent')

    return { seat: this.mySeat, gameId: joinRsp.gameId }
  }

  leave(): void {
    this.pitaya.disconnect()
  }
}
