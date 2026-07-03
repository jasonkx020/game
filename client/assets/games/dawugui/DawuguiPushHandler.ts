import { decodeProto } from './protoHelpers'
import type { PushHandler } from './PitayaClient'
import { PushRouter } from './PushRouter'

import {
  DealPush,
  TurnNotifyPush,
  PlayResultPush,
  AlertPush,
  RoundInvalidPush,
  SettlementPush,
} from '../generated/pitaya/pitaya/dawugui'
import { ErrorPush, PushHeader } from '../generated/pitaya/pitaya/common'
import { RoomStatePush } from '../generated/pitaya/pitaya/room'

export type LogFn = (line: string) => void

export function registerDawuguiPushHandlers(router: PushRouter, log: LogFn = console.log): void {
  const routes: Array<[string, (data: Uint8Array) => void]> = [
    ['onRoomState', (d) => log(formatPush('onRoomState', RoomStatePush.decode(d)))],
    ['onDeal', (d) => log(formatPush('onDeal', DealPush.decode(d)))],
    ['onTurnNotify', (d) => log(formatPush('onTurnNotify', TurnNotifyPush.decode(d)))],
    ['onPlayResult', (d) => log(formatPush('onPlayResult', PlayResultPush.decode(d)))],
    ['onAlert', (d) => log(formatPush('onAlert', AlertPush.decode(d)))],
    ['onRoundInvalid', (d) => log(formatPush('onRoundInvalid', RoundInvalidPush.decode(d)))],
    ['onSettlement', (d) => log(formatPush('onSettlement', SettlementPush.decode(d)))],
    ['onError', (d) => log(formatPush('onError', ErrorPush.decode(d)))],
  ]

  for (const [route, handler] of routes) {
    router.on(route, handler as PushHandler)
  }
}

function formatPush(name: string, msg: { header?: PushHeader | undefined }): string {
  const seq = msg.header?.meta?.actionSeq ?? 0
  return `[push:${name}] action_seq=${seq}`
}

export { DealPush, PlayResultPush, SettlementPush }
