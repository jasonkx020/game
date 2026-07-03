import { PitayaClient } from './PitayaClient'

export interface EventMeta {
  auditSn: number
  actionSeq: number
  roundId: string
  roundNo: number
  roomId: string
  serverTs: number
}

export type SyncFn = (roomId: string, roundId: string, sinceActionSeq: number) => Promise<void>

export class EventTracker {
  roundId = ''
  lastActionSeq = 0
  private syncFn?: SyncFn

  setSyncFn(fn: SyncFn): void {
    this.syncFn = fn
  }

  reset(roundId = ''): void {
    this.roundId = roundId
    this.lastActionSeq = 0
  }

  onPush(meta: EventMeta): void {
    if (this.lastActionSeq > 0 && meta.actionSeq !== this.lastActionSeq + 1) {
      void this.syncFn?.(meta.roomId, meta.roundId, this.lastActionSeq)
    }
    this.lastActionSeq = meta.actionSeq
    if (meta.roundId) this.roundId = meta.roundId
  }
}

export function attachEventTracker(client: PitayaClient, tracker: EventTracker): void {
  client.onPush('*', (_route, data) => {
    const meta = tryDecodePushMeta(data)
    if (meta) tracker.onPush(meta)
  })
}

/** Best-effort decode EventMeta from push body (field 1 nested message in PushHeader). */
function tryDecodePushMeta(data: Uint8Array): EventMeta | null {
  try {
    // PushHeader: field 1 = EventMeta (length-delimited)
    if (data.length < 2 || data[0] !== 0x0a) return null
    const len = data[1]
    const metaBytes = data.subarray(2, 2 + len)
    return decodeEventMeta(metaBytes)
  } catch {
    return null
  }
}

function decodeEventMeta(bytes: Uint8Array): EventMeta | null {
  let auditSn = 0
  let actionSeq = 0
  let roundId = ''
  let roundNo = 0
  let roomId = ''
  let serverTs = 0
  let i = 0
  while (i < bytes.length) {
    const tag = bytes[i++]
    const field = tag >> 3
    const wire = tag & 7
    if (wire === 0) {
      const { value, offset } = readVarint(bytes, i)
      i = offset
      if (field === 1) auditSn = value
      if (field === 2) actionSeq = value
      if (field === 4) roundNo = value
      if (field === 6) serverTs = value
    } else if (wire === 2) {
      const len = bytes[i++]
      const str = new TextDecoder().decode(bytes.subarray(i, i + len))
      i += len
      if (field === 3) roundId = str
      if (field === 5) roomId = str
    } else {
      break
    }
  }
  if (actionSeq === 0) return null
  return { auditSn, actionSeq, roundId, roundNo, roomId, serverTs }
}

function readVarint(data: Uint8Array, offset: number): { value: number; offset: number } {
  let id = 0
  for (let j = offset; j < data.length; j++) {
    const b = data[j]
    id += (b & 0x7f) << (7 * (j - offset))
    if (b < 128) return { value: id, offset: j + 1 }
  }
  throw new Error('bad varint')
}
