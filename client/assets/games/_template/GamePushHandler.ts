/**
 * 新游戏 Push 处理器模板 — 复制到 games/{game_id}/ 并注册 routes。
 * 见 docs/tech/client-architecture.md §9
 */
import type { PushHandler } from '../../platform/sdk/PitayaClient'
import { PushRouter } from '../../platform/sdk/PushRouter'

export function registerTemplatePushHandlers(router: PushRouter, log: (s: string) => void): void {
  router.on('onRoomState', ((_route, _data) => {
    log('[template] onRoomState')
  }) as PushHandler)
}
