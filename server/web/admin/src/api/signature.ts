const EMPTY_BODY_SHA256 =
  'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'

function canonicalQuery(search: string): string {
  if (!search || search === '?') return ''
  const params = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search)
  const keys = [...params.keys()].sort()
  const parts: string[] = []
  for (const k of keys) {
    for (const v of params.getAll(k)) {
      parts.push(`${k}=${encodeURIComponent(v)}`)
    }
  }
  parts.sort()
  return parts.join('&')
}

async function sha256Hex(data: ArrayBuffer | string): Promise<string> {
  const buf = typeof data === 'string' ? new TextEncoder().encode(data) : data
  const hash = await crypto.subtle.digest('SHA-256', buf)
  return [...new Uint8Array(hash)].map((b) => b.toString(16).padStart(2, '0')).join('')
}

async function hmacSha256Hex(secret: string, message: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const sig = await crypto.subtle.sign('HMAC', key, new TextEncoder().encode(message))
  return [...new Uint8Array(sig)].map((b) => b.toString(16).padStart(2, '0')).join('')
}

export interface SignOptions {
  method: string
  path: string
  body?: string
  token?: string
  appId: string
  appSecret: string
}

export async function signRequest(opts: SignOptions) {
  const body = opts.body ?? ''
  const contentSHA = body ? await sha256Hex(body) : EMPTY_BODY_SHA256
  const ts = Math.floor(Date.now() / 1000).toString()
  const nonce = crypto.randomUUID()
  const auth = opts.token ? `Bearer ${opts.token}` : ''
  const url = new URL(opts.path, 'http://local')
  const canonical = [
    opts.method.toUpperCase(),
    url.pathname,
    canonicalQuery(url.search),
    ts,
    nonce,
    contentSHA,
    auth,
  ].join('\n')
  const signature = await hmacSha256Hex(opts.appSecret, canonical)
  return {
    'X-App-Id': opts.appId,
    'X-Timestamp': ts,
    'X-Nonce': nonce,
    'X-Content-SHA256': contentSHA,
    'X-Signature': signature,
    ...(auth ? { Authorization: auth } : {}),
  }
}
