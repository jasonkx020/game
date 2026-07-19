import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const root = path.join(__dirname, '..', 'assets')

function walk(dir, out = []) {
  for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, ent.name)
    if (ent.isDirectory()) walk(p, out)
    else if (p.endsWith('.ts') && !p.endsWith('.d.ts')) out.push(p)
  }
  return out
}

const files = walk(root)
const importRe = /from ['"](\.[^'"]+)['"]/g
const dynamicImportRe = /import\(['"](\.[^'"]+)['"]\)/g
const issues = []

function checkImport(file, spec) {
  const base = path.dirname(file)
  const target = path.resolve(base, spec)
  const exts = ['', '.ts', '.tsx', '/index.ts']
  return exts.some((ext) => {
    const t = target + ext
    return fs.existsSync(t) && fs.statSync(t).isFile()
  })
}

for (const file of files) {
  const text = fs.readFileSync(file, 'utf8')
  for (const re of [importRe, dynamicImportRe]) {
    re.lastIndex = 0
    let m
    while ((m = re.exec(text))) {
      const spec = m[1]
      if (!checkImport(file, spec)) {
        issues.push({
          file: path.relative(path.join(__dirname, '..'), file),
          import: spec,
        })
      }
    }
  }
}

if (!issues.length) {
  console.log(`All relative imports resolve OK (${files.length} files)`)
} else {
  console.log('BROKEN IMPORTS:')
  for (const i of issues) console.log(`${i.file} -> ${i.import}`)
  process.exit(1)
}
