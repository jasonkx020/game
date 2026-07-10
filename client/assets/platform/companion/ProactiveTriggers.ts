import { SessionStore } from '../../scripts/SessionStore'
import { GameModuleRegistry } from '../lobby/GameModuleRegistry'

/** 登录欢迎、久未开局等主动触发（P2 扩展点） */
export const ProactiveTriggers = {
  onLobbyEnter(nickname: string): string {
    return nickname ? `${nickname}，欢迎回来！想聊天还是来一局？` : '欢迎回来！'
  },

  onIdleMinutes(minutes: number): string | null {
    if (minutes >= 5) return '好久没开局啦，要不要我帮你推荐一款轻松的游戏？'
    return null
  },

  onLoseStreak(count: number): string | null {
    if (count >= 3) return '连败别气馁，我陪你练练手，或者讲讲规则？'
    return null
  },
}

/** 将公开 Push 事件转为局内陪玩提示 */
export function companionHintsFromEvent(gameId: string, event: string, payload: unknown): string[] {
  const mod = GameModuleRegistry.get(gameId)
  const hints = mod?.companionHooks?.onPublicEvent?.(event, payload) ?? []
  const texts = hints.map((h) => h.text)
  for (const t of texts) SessionStore.appendCompanionHint(t)
  return texts
}
