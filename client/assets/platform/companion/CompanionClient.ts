import { signRequest } from '../sdk/signature'
import { defaultConfig, type ClientConfig } from '../sdk/config'
import { companionHintsFromEvent } from './ProactiveTriggers'

export interface CompanionSession {
  id: number
  userId: number
  personaId: string
}

export interface CompanionMessage {
  id: number
  role: string
  content: string
  toolName?: string
}

export type ChatStreamHandler = (chunk: string, done: boolean) => void

export class CompanionClient {
  private cfg: ClientConfig
  private getToken: () => string

  constructor(getToken: () => string, cfg: Partial<ClientConfig> = {}) {
    this.cfg = { ...defaultConfig, ...cfg }
    this.getToken = getToken
  }

  async createSession(personaId = 'default'): Promise<CompanionSession> {
    const res = await this.request<{ id: number; user_id: number; persona_id: string }>(
      'POST',
      '/v1/companion/sessions',
      { persona_id: personaId },
    )
    return { id: res.id, userId: res.user_id, personaId: res.persona_id }
  }

  async listMessages(sessionId: number): Promise<CompanionMessage[]> {
    const res = await this.request<{ messages: Array<Record<string, unknown>> }>(
      'GET',
      `/v1/companion/sessions/${sessionId}/messages`,
    )
    return (res.messages ?? []).map((m) => ({
      id: Number(m.id),
      role: String(m.role),
      content: String(m.content),
      toolName: m.tool_name ? String(m.tool_name) : undefined,
    }))
  }

  async chatStream(sessionId: number, message: string, onChunk: ChatStreamHandler): Promise<void> {
    const body = JSON.stringify({ message })
    const headers = await signRequest({
      method: 'POST',
      path: `/v1/companion/sessions/${sessionId}/chat`,
      body,
      token: this.getToken(),
      appId: this.cfg.appId,
      appSecret: this.cfg.appSecret,
    })
    const url = `${this.cfg.apiBaseUrl}/v1/companion/sessions/${sessionId}/chat`
    const res = await fetch(url, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json', Accept: 'text/event-stream' },
      body,
    })
    if (!res.ok || !res.body) {
      throw new Error(`companion chat failed: ${res.status}`)
    }
    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      const parts = buffer.split('\n\n')
      buffer = parts.pop() ?? ''
      for (const part of parts) {
        const line = part.trim()
        if (!line.startsWith('data: ')) continue
        const data = line.slice(6)
        if (data === '[DONE]') {
          onChunk('', true)
          return
        }
        try {
          const parsed = JSON.parse(data) as { content?: string }
          if (parsed.content) onChunk(parsed.content, false)
        } catch {
          /* ignore */
        }
      }
    }
    onChunk('', true)
  }

  submitContext(_sessionId: number, event: string, payload: unknown): void {
    const gameId = typeof payload === 'object' && payload && 'gameId' in payload
      ? String((payload as { gameId: string }).gameId)
      : ''
    if (gameId) companionHintsFromEvent(gameId, event, payload)
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const bodyStr = body !== undefined ? JSON.stringify(body) : ''
    const headers = await signRequest({
      method,
      path,
      body: bodyStr || undefined,
      token: this.getToken(),
      appId: this.cfg.appId,
      appSecret: this.cfg.appSecret,
    })
    const res = await fetch(`${this.cfg.apiBaseUrl}${path}`, {
      method,
      headers: { ...headers, ...(bodyStr ? { 'Content-Type': 'application/json' } : {}) },
      body: bodyStr || undefined,
    })
    const data = await res.json()
    if (!res.ok) throw data
    return data as T
  }
}
