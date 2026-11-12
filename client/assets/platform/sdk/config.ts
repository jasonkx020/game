export interface ClientConfig {
  apiBaseUrl: string
  /** Independent auth-service base (login/refresh/logout). Defaults to :8081. */
  authBaseUrl: string
  appId: string
  appSecret: string
  defaultWsUrl: string
  requestTimeoutMs: number
}

declare const GAME_API_BASE: string | undefined
declare const GAME_AUTH_BASE: string | undefined
declare const GAME_APP_ID: string | undefined
declare const GAME_HMAC_SECRET: string | undefined

export const defaultConfig: ClientConfig = {
  apiBaseUrl: typeof GAME_API_BASE !== 'undefined' ? GAME_API_BASE : 'http://localhost:8080',
  authBaseUrl:
    typeof GAME_AUTH_BASE !== 'undefined' ? GAME_AUTH_BASE : 'http://localhost:8081',
  appId: typeof GAME_APP_ID !== 'undefined' ? GAME_APP_ID : 'cocos-dev',
  appSecret: typeof GAME_HMAC_SECRET !== 'undefined' ? GAME_HMAC_SECRET : 'dev-hmac-secret-change-me',
  defaultWsUrl: 'ws://localhost:3250',
  requestTimeoutMs: 10000,
}
