#!/usr/bin/env node
/**
 * ts-proto emits longToNumber that throws on uint64 > MAX_SAFE_INTEGER.
 * Server snowflake audit_sn and bot user ids exceed that — patch after gen.
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const dir = path.resolve(__dirname, '../assets/platform/generated/pitaya/pitaya')

const replacement = `function longToNumber(int64: { toString(): string }): number {
  // Snowflake audit_sn / bot uid may exceed MAX_SAFE_INTEGER; do not throw.
  return globalThis.Number(int64.toString());
}`

const re =
  /function longToNumber\(int64: \{ toString\(\): string \}\): number \{[\s\S]*?\n\}/

let n = 0
for (const name of fs.readdirSync(dir)) {
  if (!name.endsWith('.ts')) continue
  const fp = path.join(dir, name)
  const src = fs.readFileSync(fp, 'utf8')
  if (!re.test(src)) continue
  const next = src.replace(re, replacement)
  if (next !== src) {
    fs.writeFileSync(fp, next)
    n++
    console.log('patched', name)
  }
}
console.log(`longToNumber patch done (${n} files)`)
