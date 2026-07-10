import { director } from 'cc'
import { SessionStore } from '../../scripts/SessionStore'
import { gameBundleManager } from './GameBundleManager'
import type { LobbyGameItem } from '../sdk/ApiClient'

export type GameHostMode = 'lobby' | 'standalone'

export interface GameHostLaunchLobby {
  mode: 'lobby'
  gameId: string
  game: LobbyGameItem
  companionSessionId?: number
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
      SessionStore.room = await SessionStore.api.createRoom({
        gameId: params.gameId,
        playerCount: params.gameId === 'liuzichong' ? 2 : 4,
        fillBots: true,
      })
      director.loadScene('Room')
      return
    }

    const game = params.game
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
    SessionStore.resetRoom()
    SessionStore.companionSessionId = params.companionSessionId ?? null
    SessionStore.room = await SessionStore.api.createRoom({
      gameId: game.gameId,
      playerCount: game.maxPlayers,
    })
    director.loadScene('Room')
  },
}
