/**
 * 创建 client/assets 下标准游戏资源目录（含 Cocos .meta）。
 * 用法：node scripts/scaffold-asset-dirs.mjs
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { randomUUID } from 'node:crypto'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const assetsRoot = path.join(__dirname, '..', 'assets')

/** @type {Record<string, string | undefined>} */
const existingMetaUuid = {}

function readExistingMeta(metaPath) {
  if (!fs.existsSync(metaPath)) return randomUUID()
  try {
    const data = JSON.parse(fs.readFileSync(metaPath, 'utf8'))
    if (data.uuid) {
      existingMetaUuid[metaPath] = data.uuid
      return data.uuid
    }
  } catch {
    /* ignore */
  }
  return randomUUID()
}

function folderMeta(uuid) {
  return `${JSON.stringify(
    {
      ver: '1.2.0',
      importer: 'directory',
      imported: true,
      uuid,
      files: [],
      subMetas: {},
      userData: {},
    },
    null,
    2,
  )}\n`
}

function ensureDir(relPath, note) {
  const dir = path.join(assetsRoot, relPath)
  fs.mkdirSync(dir, { recursive: true })
  const keep = path.join(dir, '.gitkeep')
  if (!fs.existsSync(keep)) fs.writeFileSync(keep, '')
  const metaPath = `${dir}.meta`
  const uuid = readExistingMeta(metaPath)
  fs.writeFileSync(metaPath, folderMeta(uuid))
  if (note) {
    const readme = path.join(dir, 'README.txt')
    if (!fs.existsSync(readme)) fs.writeFileSync(readme, note)
  }
}

const gameAssetNote = `用途：游戏 Bundle 资源目录
- 在 Cocos 编辑器中将 assets/games/{gameId}/ 配置为 Remote Bundle
- 通过 Inspector 拖拽引用，或 bundle.load('images/xxx/spriteFrame') 动态加载
`

const dirs = [
  ['platform/ui', '平台通用 UI 资源根目录'],
  ['platform/ui/common', '平台通用：按钮、背景、Logo 等'],
  ['platform/ui/lobby', '大厅 UI 装饰'],
  ['platform/ui/lobby/icons', '大厅游戏列表本地占位图标（可选，生产优先用 API icon_url）'],
  ['scripts/scenes/images', 'Launch / Lobby / Room 平台场景专用图片'],
  ['games/_template/images', '模板：游戏图标、小图'],
  ['games/_template/textures', '模板：牌面、棋盘、背景等大图'],
  ['games/_template/ui', '模板：游戏内按钮、面板'],
  ['games/_template/audio', '模板：音效（可选）'],
  ['games/dawugui/images', '打乌龟：图标、小图'],
  ['games/dawugui/textures', '打乌龟：牌面、桌面背景等'],
  ['games/dawugui/ui', '打乌龟：游戏内 UI'],
  ['games/dawugui/audio', '打乌龟：音效（可选）'],
  ['games/liuzichong/images', '六子冲：图标、棋子小图'],
  ['games/liuzichong/textures', '六子冲：棋盘、背景等'],
  ['games/liuzichong/ui', '六子冲：游戏内 UI'],
  ['games/liuzichong/audio', '六子冲：音效（可选）'],
]

for (const [rel, note] of dirs) {
  ensureDir(rel, note)
}

for (const gameId of ['_template', 'dawugui', 'liuzichong']) {
  const guidePath = path.join(assetsRoot, 'games', gameId, 'ASSETS.txt')
  if (!fs.existsSync(guidePath)) {
    fs.writeFileSync(
      guidePath,
      `${gameAssetNote}
images/    图标、封面、小图
textures/  牌面 / 棋盘 / 场景背景
ui/        游戏内按钮、面板
audio/     音效（可选）

脚本与场景与资源同目录：*.ts、*.scene 放在 games/${gameId}/ 根下。
`,
    )
  }
}

console.log(`Scaffolded ${dirs.length} asset directories under ${assetsRoot}`)
