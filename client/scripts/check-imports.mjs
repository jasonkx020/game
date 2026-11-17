/**
 * 1) Relative imports must resolve to an existing .ts file.
 * 2) Directory boundaries (mirrors docs/tech/client-structure.md):
 *    - platform/* must not import games/*
 *    - games/{a} must not import games/{b}
 *    - scripts/* must not import games/* (use Registry + Bundle)
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const clientRoot = path.join(__dirname, '..')
const assetsRoot = path.join(clientRoot, 'assets')

function walk(dir, out = []) {
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, ent.name)
    if (ent.isDirectory()) walk(p, out)
    else if (p.endsWith('.ts') && !p.endsWith('.d.ts')) out.push(p)
  }
  return out
}

function posixRel(fromRoot, file) {
  return path.relative(fromRoot, file).split(path.sep).join('/')
}

function zoneOf(relPosix) {
  if (relPosix.startsWith('platform/')) return { kind: 'platform' }
  if (relPosix.startsWith('scripts/')) return { kind: 'scripts' }
  const m = relPosix.match(/^games\/([^/]+)\//)
  if (m) return { kind: 'game', gameId: m[1] }
  if (relPosix.startsWith('games/')) return { kind: 'games-root' }
  return { kind: 'other' }
}

function resolveImportTarget(file, spec) {
  const base = path.dirname(file)
  const target = path.resolve(base, spec)
  const exts = ['', '.ts', '.tsx', '/index.ts']
  for (const ext of exts) {
    const t = target + ext
    if (fs.existsSync(t) && fs.statSync(t).isFile()) return t
  }
  return null
}

function collectSpecs(text) {
  const specs = []
  for (const re of [/from ['"](\.[^'"]+)['"]/g, /import\(['"](\.[^'"]+)['"]\)/g]) {
    re.lastIndex = 0
    let m
    while ((m = re.exec(text))) specs.push(m[1])
  }
  return specs
}

const files = walk(assetsRoot)
const broken = []
const boundary = []

for (const file of files) {
  const rel = posixRel(assetsRoot, file)
  const from = zoneOf(rel)
  if (from.kind === 'game' && from.gameId === '_template') continue

  const text = fs.readFileSync(file, 'utf8')
  for (const spec of collectSpecs(text)) {
    const resolved = resolveImportTarget(file, spec)
    if (!resolved) {
      broken.push({ file: posixRel(clientRoot, file), import: spec })
      continue
    }
    if (!resolved.startsWith(assetsRoot)) continue
    const toRel = posixRel(assetsRoot, resolved)
    const to = zoneOf(toRel)

    // Sole exception: GameBundleManager dynamic-imports *Module for preview/builtin fallback (ADR-006).
    const allowBuiltinFallback =
      rel === 'platform/lobby/GameBundleManager.ts' && to.kind === 'game'
    if (from.kind === 'platform' && to.kind === 'game' && !allowBuiltinFallback) {
      boundary.push({
        file: posixRel(clientRoot, file),
        import: spec,
        rule: 'platform must not import games/* (only GameBundleManager builtin fallback allowed)',
      })
    }
    if (from.kind === 'scripts' && to.kind === 'game') {
      boundary.push({
        file: posixRel(clientRoot, file),
        import: spec,
        rule: 'scripts must not import games/* (use GameModuleRegistry + Bundle)',
      })
    }
    if (from.kind === 'game' && to.kind === 'game' && from.gameId !== to.gameId) {
      if (to.gameId === '_template') continue
      boundary.push({
        file: posixRel(clientRoot, file),
        import: spec,
        rule: `games/${from.gameId} must not import games/${to.gameId}`,
      })
    }
  }
}

let failed = false
if (broken.length) {
  failed = true
  console.log('BROKEN IMPORTS:')
  for (const i of broken) console.log(`  ${i.file} -> ${i.import}`)
}
if (boundary.length) {
  failed = true
  console.log('BOUNDARY VIOLATIONS (see docs/tech/client-structure.md):')
  for (const i of boundary) console.log(`  ${i.file} -> ${i.import}  [${i.rule}]`)
}
if (!failed) {
  console.log(`All relative imports + boundaries OK (${files.length} files)`)
} else {
  process.exit(1)
}
