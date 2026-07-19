import {
  buildDataRequest,
  buildHandshakeAckPacket,
  buildHandshakePacket,
  buildHeartbeatPacket,
  decodeMessageAsync,
  decodePackets,
  MessageType,
  PacketType,
} from './PitayaPacket'

export type PushHandler = (route: string, data: Uint8Array) => void

export interface PitayaClientOptions {
  requestTimeoutMs?: number
}

export class PitayaClient {
  private ws: WebSocket | null = null
  private nextId = 1
  private pending = new Map<
    number,
    { resolve: (v: Uint8Array) => void; reject: (e: Error) => void; timer: ReturnType<typeof setTimeout> }
  >()
  private pushHandlers = new Map<string, Set<PushHandler>>()
  private buffer = new Uint8Array()
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null
  private handshakeDone = false
  private handshakeResolve: (() => void) | null = null
  private opts: PitayaClientOptions

  constructor(opts: PitayaClientOptions = {}) {
    this.opts = opts
  }

  async connect(wsUrl: string): Promise<void> {
    if (this.ws) this.disconnect()

    await new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(wsUrl)
      ws.binaryType = 'arraybuffer'
      this.handshakeResolve = resolve

      ws.onopen = () => {
        ws.send(buildHandshakePacket())
      }
      ws.onerror = () => reject(new Error('websocket error'))
      ws.onclose = () => {
        if (!this.handshakeDone) reject(new Error('websocket closed before handshake'))
      }
      ws.onmessage = (ev) => {
        this.handleRawMessage(new Uint8Array(ev.data as ArrayBuffer))
      }
      this.ws = ws
    })
  }

  disconnect(): void {
    if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)
    this.heartbeatTimer = null
    for (const p of this.pending.values()) {
      clearTimeout(p.timer)
      p.reject(new Error('disconnected'))
    }
    this.pending.clear()
    this.ws?.close()
    this.ws = null
    this.handshakeDone = false
    this.handshakeResolve = null
    this.buffer = new Uint8Array()
  }

  onPush(route: string, handler: PushHandler): void {
    if (!this.pushHandlers.has(route)) this.pushHandlers.set(route, new Set())
    this.pushHandlers.get(route)!.add(handler)
  }

  offPush(route: string, handler: PushHandler): void {
    this.pushHandlers.get(route)?.delete(handler)
  }

  async request(route: string, data: Uint8Array): Promise<Uint8Array> {
    if (!this.ws || !this.handshakeDone) throw new Error('not connected')
    const id = this.nextId++
    const packet = buildDataRequest(id, route, data)
    const timeoutMs = this.opts.requestTimeoutMs ?? 10000
    return new Promise<Uint8Array>((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id)
        reject(new Error(`request timeout: ${route}`))
      }, timeoutMs)
      this.pending.set(id, { resolve, reject, timer })
      this.ws!.send(packet)
    })
  }

  private startHeartbeat(intervalSec: number): void {
    if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)
    this.heartbeatTimer = setInterval(() => {
      this.ws?.send(buildHeartbeatPacket())
    }, intervalSec * 1000)
  }

  private completeHandshake(): void {
    if (this.handshakeDone) return
    this.handshakeDone = true
    this.ws?.send(buildHandshakeAckPacket())
    this.startHeartbeat(15)
    this.handshakeResolve?.()
    this.handshakeResolve = null
  }

  private handleRawMessage(chunk: Uint8Array): void {
    const merged = new Uint8Array(this.buffer.length + chunk.length)
    merged.set(this.buffer)
    merged.set(chunk, this.buffer.length)
    const { packets, rest } = decodePackets(merged)
    this.buffer = rest
    void this.dispatchPackets(packets)
  }

  private async dispatchPackets(
    packets: { type: PacketType; data: Uint8Array }[],
  ): Promise<void> {
    for (const pkt of packets) {
      switch (pkt.type) {
        case PacketType.Handshake:
          if (!this.handshakeDone) {
            try {
              const sys = JSON.parse(new TextDecoder().decode(pkt.data)) as {
                code?: number
                sys?: { heartbeat?: number }
              }
              const hb = sys.sys?.heartbeat ?? 15
              this.completeHandshake()
              if (this.heartbeatTimer) clearInterval(this.heartbeatTimer)
              this.startHeartbeat(hb)
            } catch {
              this.completeHandshake()
            }
          }
          break
        case PacketType.Heartbeat:
          this.ws?.send(buildHeartbeatPacket())
          break
        case PacketType.Data:
          await this.handleDataPacket(pkt.data)
          break
        case PacketType.Kick:
          this.disconnect()
          break
      }
    }
  }

  private async handleDataPacket(data: Uint8Array): Promise<void> {
    const msg = await decodeMessageAsync(data)
    if (msg.type === MessageType.Response) {
      const p = this.pending.get(msg.id)
      if (p) {
        clearTimeout(p.timer)
        this.pending.delete(msg.id)
        if (msg.err) {
          p.reject(new Error(new TextDecoder().decode(msg.data)))
        } else {
          p.resolve(msg.data)
        }
      }
      return
    }
    if (msg.type === MessageType.Push) {
      const handlers = this.pushHandlers.get(msg.route)
      if (handlers) {
        for (const h of handlers) h(msg.route, msg.data)
      }
      const anyHandlers = this.pushHandlers.get('*')
      if (anyHandlers) {
        for (const h of anyHandlers) h(msg.route, msg.data)
      }
    }
  }
}
