import { parse as parseYaml } from 'yaml'

/** JSON Pointer get; pointer like `/UserProfile` or `/~1v1~1user~1profile`. */
function pointerGet(doc: unknown, pointer: string): unknown {
  if (!pointer || pointer === '/') return doc
  const raw = pointer.startsWith('/') ? pointer.slice(1) : pointer
  if (!raw) return doc
  const parts = raw.split('/').map((p) => p.replace(/~1/g, '/').replace(/~0/g, '~'))
  let cur: unknown = doc
  for (const p of parts) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[p]
  }
  return cur
}

function splitRef(ref: string): { pathPart: string; pointer: string } {
  const hash = ref.indexOf('#')
  if (hash < 0) return { pathPart: ref, pointer: '' }
  return { pathPart: ref.slice(0, hash), pointer: ref.slice(hash + 1) }
}

function resolveRefUrl(fromUrl: string, refPath: string): string {
  if (!refPath) return fromUrl.split('#')[0]
  if (refPath.startsWith('/')) {
    return new URL(refPath, window.location.origin).href
  }
  return new URL(refPath, fromUrl).href
}

/**
 * Fetch a multi-file OpenAPI YAML entry and deeply inline all `$ref`s.
 * Avoids swagger-ui's broken relative-ref resolution across path fragments.
 */
export async function bundleOpenApi(entryUrl: string): Promise<Record<string, unknown>> {
  const rootUrl = new URL(entryUrl, window.location.href).href.split('#')[0]
  const docCache = new Map<string, unknown>()

  async function loadDoc(absUrl: string): Promise<unknown> {
    const key = absUrl.split('#')[0]
    if (docCache.has(key)) return docCache.get(key)
    const res = await fetch(key)
    if (!res.ok) throw new Error(`无法加载 OpenAPI 片段 ${key}（${res.status}）`)
    const text = await res.text()
    const doc = parseYaml(text)
    docCache.set(key, doc)
    return doc
  }

  async function deref(node: unknown, baseUrl: string, stack: Set<string>): Promise<unknown> {
    if (node == null || typeof node !== 'object') return node
    if (Array.isArray(node)) {
      return Promise.all(node.map((n) => deref(n, baseUrl, stack)))
    }

    const obj = node as Record<string, unknown>
    if (typeof obj.$ref === 'string') {
      const { pathPart, pointer } = splitRef(obj.$ref)
      const targetUrl = resolveRefUrl(baseUrl, pathPart)
      const ptr = pointer.startsWith('/') ? pointer : pointer ? `/${pointer}` : ''
      const stackKey = `${targetUrl}#${ptr}`
      if (stack.has(stackKey)) {
        return { $ref: obj.$ref }
      }
      stack.add(stackKey)
      const doc = await loadDoc(targetUrl)
      const target = pointerGet(doc, ptr)
      if (target === undefined) {
        stack.delete(stackKey)
        throw new Error(`无法解析 $ref: ${obj.$ref}（自 ${baseUrl}）`)
      }
      const resolved = await deref(structuredClone(target), targetUrl, stack)
      stack.delete(stackKey)
      const { $ref: _drop, ...siblings } = obj
      if (Object.keys(siblings).length === 0) return resolved
      if (resolved && typeof resolved === 'object' && !Array.isArray(resolved)) {
        return { ...(resolved as object), ...siblings }
      }
      return resolved
    }

    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(obj)) {
      out[k] = await deref(v, baseUrl, stack)
    }
    return out
  }

  const root = await loadDoc(rootUrl)
  return (await deref(root, rootUrl, new Set())) as Record<string, unknown>
}
