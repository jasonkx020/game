import type { PushHandler } from '../../platform/sdk/PitayaClient'
import { PushRouter } from '../../platform/sdk/PushRouter'

import {
  DealPush,
  TurnNotifyPush,
  PlayResultPush,
  AlertPush,
  RoundInvalidPush,
  SettlementPush,
} from '../../platform/generated/pitaya/pitaya/dawugui'
import { ErrorPush, PushHeader } from '../../platform/generated/pitaya/pitaya/common'
import { RoomStatePush } from '../../platform/generated/pitaya/pitaya/room'

export type LogFn = (line: string) => void

export function registerDawuguiPushHandlers(router: PushRouter, log: LogFn = console.log): void {
  const routes: Array<[string, PushHandler]> = [
    ['onRoomState', (_route, data) => log(formatPush('onRoomState', RoomStatePush.decode(data)))],
    ['onDeal', (_route, data) => log(formatPush('onDeal', DealPush.decode(data)))],
    ['onTurnNotify', (_route, data) => log(formatPush('onTurnNotify', TurnNotifyPush.decode(data)))],
    ['onPlayResult', (_route, data) => log(formatPush('onPlayResult', PlayResultPush.decode(data)))],
    ['onAlert', (_route, data) => log(formatPush('onAlert', AlertPush.decode(data)))],
    ['onRoundInvalid', (_route, data) => log(formatPush('onRoundInvalid', RoundInvalidPush.decode(data)))],
    ['onSettlement', (_route, data) => log(formatPush('onSettlement', SettlementPush.decode(data)))],
    ['onError', (_route, data) => log(formatPush('onError', ErrorPush.decode(data)))],
  ]

  for (const [route, handler] of routes) {
    router.on(route, handler)
  }
}

function formatPush(name: string, msg: { header?: PushHeader | undefined }): string {
  const seq = msg.header?.meta?.actionSeq ?? 0
  return `[push:${name}] action_seq=${seq}`
}

export { DealPush, PlayResultPush, SettlementPush }
