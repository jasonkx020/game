import { Color, Graphics, Label, Layers, Node, UIOpacity, UITransform, Vec3 } from 'cc'

/** 全局设计分辨率（竖屏） */
export const UI_DESIGN_WIDTH = 720
export const UI_DESIGN_HEIGHT = 1280

/** Canvas UI 相机只渲染 UI_2D；代码 new Node 默认 DEFAULT，不设层则不可见 */
export function ensureUILayer(node: Node): void {
  node.layer = Layers.Enum.UI_2D
  for (const child of node.children) ensureUILayer(child)
}

export function createUINode(name: string): Node {
  const node = new Node(name)
  node.layer = Layers.Enum.UI_2D
  return node
}

/**
 * 严格取自登录 HTML 主色（勿改 hex）：
 * body #e8f4fd | 容器渐变 #f0f9ff→#d9edf7 | 强调 #3b82f6/#60a5fa
 * logo #1a4b6d | 正文 #1a2a3a | muted #4a6a80 | label #2c5775
 * gold/goldDim = 品牌蓝（主按钮/链接）
 */
export const LobbyTheme = {
  /** body background #e8f4fd */
  bgDeep: new Color(0xe8, 0xf4, 0xfd, 255),
  /** #login-app gradient 0% #f0f9ff */
  bgMid: new Color(0xf0, 0xf9, 0xff, 255),
  /** #login-app gradient 100% #d9edf7 */
  bgSoft: new Color(0xd9, 0xed, 0xf7, 255),
  /** ::before rgba(147,197,253,0.3) */
  bgGlowA: new Color(147, 197, 253, 77),
  /** ::after rgba(191,219,254,0.25) */
  bgGlowB: new Color(191, 219, 254, 64),
  /** .forgot-link / .login-btn end #3b82f6 */
  gold: new Color(0x3b, 0x82, 0xf6, 255),
  /** .login-btn start #60a5fa */
  goldDim: new Color(0x60, 0xa5, 0xfa, 255),
  /** .logo-text #1a4b6d */
  brand: new Color(0x1a, 0x4b, 0x6d, 255),
  /** body/input color #1a2a3a */
  text: new Color(0x1a, 0x2a, 0x3a, 255),
  /** .subtitle / .form-options #4a6a80 */
  textMuted: new Color(0x4a, 0x6a, 0x80, 255),
  /** .input-group label / .social-item #2c5775 */
  textLabel: new Color(0x2c, 0x57, 0x75, 255),
  /** input::placeholder #9bb7cc */
  textFaint: new Color(0x9b, 0xb7, 0xcc, 255),
  /** .input-icon / .toast-sub #6b8fa0 */
  iconMuted: new Color(0x6b, 0x8f, 0xa0, 255),
  /** .social-login .divider #8bb0c8 */
  divider: new Color(0x8b, 0xb0, 0xc8, 255),
  /** 白卡片：对齐 input-wrap / social rgba(255,255,255,0.7~0.95) */
  cardBg: new Color(255, 255, 255, 230),
  surface: new Color(255, 255, 255, 242),
  panel: new Color(255, 255, 255, 153),
  inputBg: new Color(255, 255, 255, 178),
  barBg: new Color(255, 255, 255, 230),
  /** .toast rgba(255,255,255,0.95) + color #1a2a3a */
  toastBg: new Color(255, 255, 255, 242),
  toastText: new Color(0x1a, 0x2a, 0x3a, 255),
  btnText: new Color(255, 255, 255, 255),
  online: new Color(0x22, 0xc5, 0x5e, 255),
  danger: new Color(0xef, 0x44, 0x44, 255),
  /** 房卡/金币数字用品牌蓝（对齐大厅 HTML .game-coin / .room-card-balance） */
  coin: new Color(0x3b, 0x82, 0xf6, 255),
  /** .game-card[data-game] 柔和底色 rgba(*,0.08)≈alpha 20 */
  cardTints: [
    new Color(239, 68, 68, 20),
    new Color(59, 130, 246, 20),
    new Color(251, 146, 60, 20),
    new Color(34, 197, 94, 20),
    new Color(6, 182, 212, 20),
    new Color(168, 85, 247, 20),
    new Color(236, 72, 153, 20),
    new Color(251, 191, 36, 20),
  ],
  /** 房卡区背景 rgba(59,130,246,0.1) */
  roomCardBg: new Color(59, 130, 246, 26),
  /** 状态栏 rgba(255,255,255,0.6) */
  statusBarBg: new Color(255, 255, 255, 153),
}

/**
 * 对齐 #login-app：linear-gradient(145deg, #f0f9ff → #d9edf7) + 双光晕。
 */
export function paintSkyBackground(parent: Node, w: number, h: number): void {
  // 底层偏 #d9edf7（渐变终点）
  const bg = createUINode('Bg')
  bg.addComponent(UITransform).setContentSize(w, h)
  paintRoundRect(bg, w, h, LobbyTheme.bgSoft, 0)
  place(bg, 0, 0)
  parent.addChild(bg)

  // 上半 #f0f9ff（渐变起点）
  const mid = createUINode('BgMid')
  mid.addComponent(UITransform).setContentSize(w, h * 0.62)
  paintRoundRect(mid, w, h * 0.62, LobbyTheme.bgMid, 0)
  place(mid, 0, h * 0.19)
  parent.addChild(mid)

  // 外缘略透 #e8f4fd（body）
  const edge = createUINode('BgEdge')
  edge.addComponent(UITransform).setContentSize(w, h * 0.18)
  paintRoundRect(edge, w, h * 0.18, LobbyTheme.bgDeep, 0)
  place(edge, 0, -h * 0.41)
  parent.addChild(edge)

  const glow1 = createUINode('GlowA')
  glow1.addComponent(UITransform).setContentSize(280, 280)
  paintRoundRect(glow1, 280, 280, LobbyTheme.bgGlowA, 140)
  place(glow1, w * 0.32, h * 0.4)
  parent.addChild(glow1)

  const glow2 = createUINode('GlowB')
  glow2.addComponent(UITransform).setContentSize(240, 240)
  paintRoundRect(glow2, 240, 240, LobbyTheme.bgGlowB, 120)
  place(glow2, -w * 0.3, -h * 0.36)
  parent.addChild(glow2)
}

export const GAME_ICONS: Record<string, string> = {
  dawugui: '🐢',
  liuzichong: '♟️',
}

export function gameIcon(gameId: string, fallback = '🎮'): string {
  return GAME_ICONS[gameId] || fallback
}

export function formatCoin(n: number): string {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  return String(n)
}

export function createLabelNode(
  name: string,
  text: string,
  fontSize: number,
  color: Color,
  width = 200,
  height = 28,
): Node {
  const node = createUINode(name)
  node.addComponent(UITransform).setContentSize(width, height)
  const label = node.addComponent(Label)
  label.string = text
  label.fontSize = fontSize
  label.color = color
  label.overflow = Label.Overflow.CLAMP
  label.horizontalAlign = Label.HorizontalAlign.LEFT
  label.verticalAlign = Label.VerticalAlign.CENTER
  return node
}

export function paintRoundRect(node: Node, w: number, h: number, color: Color, radius = 12): Graphics {
  let g = node.getComponent(Graphics)
  if (!g) g = node.addComponent(Graphics)
  g.clear()
  g.fillColor = color
  g.roundRect(-w / 2, -h / 2, w, h, radius)
  g.fill()
  return g
}

export function clearChildren(root: Node | null): void {
  if (!root) return
  root.removeAllChildren()
}

export function setNodeOpacity(node: Node, opacity: number): void {
  let op = node.getComponent(UIOpacity)
  if (!op) op = node.addComponent(UIOpacity)
  op.opacity = opacity
}

export function place(node: Node, x: number, y: number): void {
  node.setPosition(new Vec3(x, y, 0))
}
