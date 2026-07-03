import type { PushHandler } from './PitayaClient'

export class PushRouter {
  private handlers = new Map<string, Set<PushHandler>>()

  on(route: string, handler: PushHandler): void {
    if (!this.handlers.has(route)) this.handlers.set(route, new Set())
    this.handlers.get(route)!.add(handler)
  }

  off(route: string, handler: PushHandler): void {
    this.handlers.get(route)?.delete(handler)
  }

  dispatch(route: string, data: Uint8Array): void {
    const set = this.handlers.get(route)
    if (set) {
      for (const h of set) h(route, data)
    }
  }

  bindToClient(client: { onPush(route: string, handler: PushHandler): void }): void {
    for (const route of this.handlers.keys()) {
      for (const handler of this.handlers.get(route)!) {
        client.onPush(route, handler)
      }
    }
  }
}
