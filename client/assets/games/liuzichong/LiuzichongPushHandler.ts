import type { PushHandler } from '../../platform/sdk/PitayaClient'
import { PushRouter } from '../../platform/sdk/PushRouter'

import {
  BoardInitPush,
  MoveResultPush,
} from '../../platform/generated/pitaya/pitaya/liuzichong'
import { TurnNotifyPush, SettlementPush } from '../../platform/generated/pitaya/pitaya/dawugui'
import { ErrorPush, PushHeader } from '../../platform/generated/pitaya/pitaya/common'
import { RoomStatePush } from '../../platform/generated/pitaya/pitaya/room'

export type LogFn = (line: string) => void
export type StateListener = () => void

const BOARD_SIZE = 16

/** 六子冲 Push 驱动的共享状态（跨场景保留） */
export class LiuzichongState {
  board: number[] = new Array(BOARD_SIZE).fill(0)
  currentSeat = 0
  mySeat = -1
  gameOver = false
  winnerSeat = -1
  private listeners = new Set<StateListener>()

  reset(): void {
    this.board = new Array(BOARD_SIZE).fill(0)
    this.currentSeat = 0
    this.mySeat = -1
    this.gameOver = false
    this.winnerSeat = -1
    this.notify()
  }

  onChange(fn: StateListener): () => void {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  notify(): void {
    for (const fn of this.listeners) fn()
  }
}

export const liuzichongState = new LiuzichongState()

export interface LiuzichongPushOptions {
  log?: LogFn
  userId?: number
}

export function registerLiuzichongPushHandlers(
  router: PushRouter,
  opts: LiuzichongPushOptions = {},
): void {
  const log = opts.log ?? console.log
  const userId = opts.userId ?? 0

  const routes: Array<[string, PushHandler]> = [
    [
      'onRoomState',
      (_route, data) => {
        const msg = RoomStatePush.decode(data)
        log(formatPush('onRoomState', msg))
        if (userId > 0 && msg.players) {
          for (const p of msg.players) {
            if (p.userId === BigInt(userId) || Number(p.userId) === userId) {
              liuzichongState.mySeat = p.seat
              liuzichongState.notify()
              break
            }
          }
        }
      },
    ],
    [
      'onBoardInit',
      (_route, data) => {
        const msg = BoardInitPush.decode(data)
        log(formatPush('onBoardInit', msg))
        liuzichongState.board = [...msg.cells]
        liuzichongState.currentSeat = msg.firstSeat
        liuzichongState.gameOver = false
        liuzichongState.notify()
      },
    ],
    [
      'onTurnNotify',
      (_route, data) => {
        const msg = TurnNotifyPush.decode(data)
        log(formatPush('onTurnNotify', msg))
        liuzichongState.currentSeat = msg.currentSeat
        liuzichongState.notify()
      },
    ],
    [
      'onMoveResult',
      (_route, data) => {
        const msg = MoveResultPush.decode(data)
        log(formatPush('onMoveResult', msg))
        const idxFrom = msg.fromRow * 4 + msg.fromCol
        const idxTo = msg.toRow * 4 + msg.toCol
        if (idxFrom >= 0 && idxFrom < BOARD_SIZE) liuzichongState.board[idxFrom] = 0
        if (idxTo >= 0 && idxTo < BOARD_SIZE) {
          liuzichongState.board[idxTo] = msg.seat + 1
        }
        for (const c of msg.captured) {
          const idx = c.row * 4 + c.col
          if (idx >= 0 && idx < BOARD_SIZE) liuzichongState.board[idx] = 0
        }
        if (msg.nextSeat !== undefined && msg.nextSeat !== null) {
          liuzichongState.currentSeat = msg.nextSeat
        }
        liuzichongState.notify()
      },
    ],
    [
      'onSettlement',
      (_route, data) => {
        const msg = SettlementPush.decode(data)
        log(formatPush('onSettlement', msg))
        liuzichongState.gameOver = true
        if (msg.scores) {
          let bestSeat = -1
          let bestScore = -999
          for (const s of msg.scores) {
            if (s.ruleScore > bestScore) {
              bestScore = s.ruleScore
              bestSeat = s.seat
            }
          }
          liuzichongState.winnerSeat = bestSeat
        }
        liuzichongState.notify()
      },
    ],
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
