import { ApiClient } from './ApiClient'
import { defaultConfig } from './config'
import { attachEventTracker, EventTracker } from './EventTracker'
import { PitayaClient } from './PitayaClient'
import { encodeProto, decodeProto } from './protoHelpers'
import { PushRouter } from './PushRouter'

import { EntryReq, EntryRsp, BindReq, BindRsp } from '../generated/pitaya/pitaya/connector'
import { JoinReq, JoinRsp, ReadyReq, ReadyRsp, SyncReq, SyncRsp } from '../generated/pitaya/pitaya/room'

const DEVICE_ID_KEY = 'games.device_id'
/** Max times to call assign after a failed connector (exclude_id). */
const MAX_ASSIGN_ROUNDS = 3

export interface EnterRoomParams {
  /** Fallback when discovery assign is unavailable (e.g. createRoom ws_url). */
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
  /** Last assigned connector id (for exclude on retry). */
  assignedConnectorId = ''

  constructor(opts: GameSessionOptions = {}) {
    this.api = opts.api ?? new ApiClient()
    this.log = opts.onLog ?? ((s) => console.log(s))
  }

  get apiClient(): ApiClient {
    return this.api
  }

  /**
   * Connect using discovery assign: gateway → assigned connector → re-assign
   * (limited) → fallback wsUrl / defaultWsUrl. Never probes a full node list.
   */
  async connectAssigned(fallbackWsUrl: string): Promise<void> {
    let excludeId: string | undefined
    let triedGateway = false
    let lastErr: Error | undefined

    for (let round = 0; round < MAX_ASSIGN_ROUNDS; round++) {
      let assign: Awaited<ReturnType<ApiClient['assignConnector']>> | null = null
      try {
        assign = await this.api.assignConnector(excludeId)
      } catch (e) {
        this.log(`[discovery] assign failed: ${e instanceof Error ? e.message : String(e)}`)
      }

      const attempts: Array<{ url: string; connectorId?: string; label: string }> = []
      if (assign?.wsGatewayUrl && !triedGateway) {
        attempts.push({ url: assign.wsGatewayUrl, label: 'gateway' })
        triedGateway = true
      }
      if (assign?.connector?.wsUrl) {
        attempts.push({
          url: assign.connector.wsUrl,
          connectorId: assign.connector.id,
          label: `connector:${assign.connector.id}`,
        })
      }

      for (const a of attempts) {
        try {
          this.log(`[pitaya] trying ${a.label} ${a.url}`)
          await this.pitaya.connect(a.url)
          this.assignedConnectorId = a.connectorId ?? ''
          this.log(`[pitaya] connected via ${a.label}`)
          return
        } catch (e) {
          lastErr = e instanceof Error ? e : new Error(String(e))
          this.pitaya.disconnect()
          if (a.connectorId) {
            excludeId = a.connectorId
          }
          this.log(`[pitaya] ${a.label} failed: ${lastErr.message}`)
        }
      }

      if (!assign?.connector && attempts.length === 0) {
        break
      }
    }

    const fallback = fallbackWsUrl || defaultConfig.defaultWsUrl
    this.log(`[pitaya] fallback ${fallback}`)
    try {
      await this.pitaya.connect(fallback)
      this.assignedConnectorId = ''
    } catch (e) {
      throw lastErr ?? (e instanceof Error ? e : new Error(String(e)))
    }
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

    await this.connectAssigned(params.wsUrl)

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
