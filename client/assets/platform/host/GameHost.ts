import { director } from 'cc'
import { SessionStore } from '../../scripts/SessionStore'
import { gameBundleManager } from '../lobby/GameBundleManager'
import type { LobbyGameItem } from '../sdk/ApiClient'

export type GameHostMode = 'lobby' | 'standalone'

export interface GameHostLaunchLobby {
  mode: 'lobby'
  gameId: string
  game: LobbyGameItem
  companionSessionId?: number
  /** 加入已有房间（真人匹配） */
  joinRoomId?: string
  /** 是否用人机补位；默认六子冲 true，其它 false */
  fillBots?: boolean
}

export interface GameHostLaunchStandalone {
  mode: 'standalone'
  gameId: string
}

export type GameHostLaunch = GameHostLaunchLobby | GameHostLaunchStandalone

export const GameHost = {
  async launch(params: GameHostLaunch): Promise<void> {
    if (params.mode === 'standalone') {
      await gameBundleManager.ensureLoaded(params.gameId, {
        version: 'builtin',
        url: '',
        sizeBytes: 1,
      })
      if (!SessionStore.login) {
        director.loadScene('Launch')
        return
      }
      SessionStore.resetRoom()
      SessionStore.mySeat = -1
      SessionStore.room = await SessionStore.api.createRoom({
        gameId: params.gameId,
        playerCount: params.gameId === 'liuzichong' ? 2 : 4,
        fillBots: true,
      })
      director.loadScene('Room')
      return
    }

    const game = params.game
    try {
      if (game.bundle) {
        await gameBundleManager.ensureLoaded(game.gameId, {
          version: game.bundle.version,
          url: game.bundle.url,
          sizeBytes: game.bundle.sizeBytes,
          sha256: game.bundle.sha256,
          entryScene: game.bundle.entryScene,
        })
      } else {
        await gameBundleManager.ensureLoaded(game.gameId, {
          version: 'builtin',
          url: '',
          sizeBytes: 1,
        })
      }
    } catch (e) {
      console.error('[GameHost] ensureLoaded failed', e)
      throw e
    }
    SessionStore.resetRoom()
    SessionStore.mySeat = -1
    SessionStore.companionSessionId = params.companionSessionId ?? null

    if (params.joinRoomId) {
      const joined = await SessionStore.api.joinRoom(params.joinRoomId)
      SessionStore.room = {
        roomId: params.joinRoomId,
        wsUrl: joined.wsUrl,
        gameId: game.gameId,
        entryScene: game.bundle?.entryScene || (game.gameId === 'liuzichong' ? 'Liuzichong' : undefined),
      }
      director.loadScene('Room')
      return
    }

    const fillBots =
      params.fillBots ?? (game.gameId === 'liuzichong')
    SessionStore.room = await SessionStore.api.createRoom({
      gameId: game.gameId,
      playerCount: game.maxPlayers,
      fillBots,
    })
    if (game.bundle?.entryScene) {
      SessionStore.room.entryScene = game.bundle.entryScene
    } else if (game.gameId === 'liuzichong') {
      SessionStore.room.entryScene = 'Liuzichong'
    }
    director.loadScene('Room')
  },
}
