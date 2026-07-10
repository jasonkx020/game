import { GameHost } from '../../platform/host/GameHost'

/** 打乌龟独立 App 冷启动入口 */
export async function bootstrapStandalone(): Promise<void> {
  await GameHost.launch({ mode: 'standalone', gameId: 'dawugui' })
}
