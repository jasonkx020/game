import { Color, EditBox, EventTouch, Label, Node, UITransform } from 'cc'
import { CompanionPanel } from '../companion/CompanionPanel'
import { GameShelf } from './GameShelf'
import {
  LobbyTheme,
  UI_DESIGN_HEIGHT,
  UI_DESIGN_WIDTH,
  createLabelNode,
  createUINode,
  ensureUILayer,
  paintRoundRect,
  paintSkyBackground,
  place,
} from './LobbyUIKit'

export interface LobbyBuiltUI {
  root: Node
  userLabel: Label
  balanceLabel: Label
  onlineLabel: Label
  statusLabel: Label
  toastRoot: Node
  toastLabel: Label
  toastSubLabel: Label
  avatarMenuRoot: Node
  companionPanelRoot: Node
  companionBadgeLabel: Label
  clubLabel: Label
  gameShelf: GameShelf
  companionPanel: CompanionPanel
  matchPanelRoot: Node
  matchRoomInput: EditBox
  matchTitleLabel: Label
}

type Handlers = {
  onUserInfo: () => void
  onRoomCard: () => void
  onCompanion: () => void
  onProfile: () => void
  onLogout: () => void
  onEditGames: () => void
  onClub: () => void
  onMatchVsBot: () => void
  onMatchCreatePvp: () => void
  onMatchJoin: () => void
  onMatchClose: () => void
}

const W = UI_DESIGN_WIDTH
const H = UI_DESIGN_HEIGHT
const PAD = 16

/**
 * 对齐大厅 HTML：顶栏 / 游戏架标题 / 4 列网格 / 推荐横滑 / 最近标签 / 俱乐部底栏。
 */
export function buildLobbyUI(parent: Node, handlers: Handlers): LobbyBuiltUI {
  const old = parent.getChildByName('AutoLobby')
  if (old) old.destroy()

  const root = createUINode('AutoLobby')
  root.addComponent(UITransform).setContentSize(W, H)
  parent.addChild(root)
  paintSkyBackground(root, W, H)

  // ===== 顶栏 status-bar =====
  const statusBar = createUINode('StatusBar')
  statusBar.addComponent(UITransform).setContentSize(W, 72)
  paintRoundRect(statusBar, W, 72, LobbyTheme.statusBarBg, 0)
  place(statusBar, 0, H / 2 - 36)
  root.addChild(statusBar)

  const userInfo = createUINode('UserInfo')
  userInfo.addComponent(UITransform).setContentSize(160, 52)
  place(userInfo, -W / 2 + 100, 0)
  statusBar.addChild(userInfo)

  const avatarBg = createUINode('AvatarBg')
  avatarBg.addComponent(UITransform).setContentSize(44, 44)
  paintRoundRect(avatarBg, 44, 44, new Color(0xbf, 0xdb, 0xfe, 255), 22)
  place(avatarBg, -50, 0)
  userInfo.addChild(avatarBg)
  const avatar = createLabelNode('Avatar', '👤', 22, LobbyTheme.brand, 40, 40)
  avatar.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(avatar, 0, 0)
  avatarBg.addChild(avatar)

  const userNameNode = createLabelNode('UserName', '玩家', 16, LobbyTheme.text, 80, 24)
  userNameNode.getComponent(Label)!.isBold = true
  place(userNameNode, 30, 10)
  userInfo.addChild(userNameNode)

  const onlineNode = createLabelNode('Online', '● 在线', 10, LobbyTheme.textMuted, 72, 18)
  place(onlineNode, 30, -12)
  userInfo.addChild(onlineNode)
  userInfo.on(Node.EventType.TOUCH_END, handlers.onUserInfo)

  const avatarMenu = createUINode('AvatarMenu')
  avatarMenu.addComponent(UITransform).setContentSize(150, 100)
  paintRoundRect(avatarMenu, 150, 100, LobbyTheme.surface, 14)
  place(avatarMenu, -W / 2 + 100, H / 2 - 120)
  avatarMenu.active = false
  root.addChild(avatarMenu)
  const profileItem = createMenuItem('ProfileItem', '👤 个人资料', LobbyTheme.text, handlers.onProfile)
  place(profileItem, 0, 22)
  avatarMenu.addChild(profileItem)
  const logoutItem = createMenuItem('LogoutItem', '🚪 退出登录', LobbyTheme.danger, handlers.onLogout)
  place(logoutItem, 0, -22)
  avatarMenu.addChild(logoutItem)

  const roomCard = createUINode('RoomCard')
  roomCard.addComponent(UITransform).setContentSize(130, 40)
  paintRoundRect(roomCard, 130, 40, LobbyTheme.roomCardBg, 20)
  place(roomCard, 20, 0)
  statusBar.addChild(roomCard)
  const coinIcon = createLabelNode('CoinIcon', '🪙', 16, LobbyTheme.coin, 28, 28)
  coinIcon.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(coinIcon, -42, 0)
  roomCard.addChild(coinIcon)
  const balNode = createLabelNode('Balance', '0', 18, LobbyTheme.coin, 48, 28)
  balNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  balNode.getComponent(Label)!.isBold = true
  place(balNode, 4, 0)
  roomCard.addChild(balNode)
  const balHint = createLabelNode('BalHint', '房卡', 10, LobbyTheme.iconMuted, 36, 18)
  place(balHint, 48, 0)
  roomCard.addChild(balHint)
  roomCard.on(Node.EventType.TOUCH_END, handlers.onRoomCard)

  const companionBtn = createUINode('CompanionEntry')
  companionBtn.addComponent(UITransform).setContentSize(40, 40)
  paintRoundRect(companionBtn, 40, 40, LobbyTheme.panel, 20)
  place(companionBtn, W / 2 - 40, 0)
  statusBar.addChild(companionBtn)
  const heart = createLabelNode('Heart', '❤️', 18, LobbyTheme.text, 36, 36)
  heart.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  companionBtn.addChild(heart)
  const badgeNode = createLabelNode('Badge', '●', 10, LobbyTheme.online, 16, 16)
  place(badgeNode, 12, 12)
  companionBtn.addChild(badgeNode)
  companionBtn.on(Node.EventType.TOUCH_END, handlers.onCompanion)

  // ===== 游戏架标题 =====
  const shelfHeader = createUINode('ShelfHeader')
  shelfHeader.addComponent(UITransform).setContentSize(W - PAD * 2, 36)
  place(shelfHeader, 0, H / 2 - 100)
  root.addChild(shelfHeader)
  const shelfTitle = createLabelNode('ShelfTitle', '🎮 游戏架 · 置顶优先', 17, LobbyTheme.text, 280, 30)
  shelfTitle.getComponent(Label)!.isBold = true
  place(shelfTitle, -W / 2 + PAD + 140, 0)
  shelfHeader.addChild(shelfTitle)
  const editBtn = createLabelNode('EditGames', '编辑', 12, LobbyTheme.gold, 48, 28)
  editBtn.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.RIGHT
  place(editBtn, W / 2 - PAD - 24, 0)
  shelfHeader.addChild(editBtn)
  editBtn.on(Node.EventType.TOUCH_END, handlers.onEditGames)

  // ===== 游戏网格（4 列）=====
  const gridRoot = createUINode('GridRoot')
  gridRoot.addComponent(UITransform).setContentSize(W - PAD * 2, 230)
  place(gridRoot, 0, H / 2 - 250)
  root.addChild(gridRoot)

  // ===== 推荐区 =====
  const recHeader = createUINode('RecHeader')
  recHeader.addComponent(UITransform).setContentSize(W - PAD * 2, 32)
  place(recHeader, 0, H / 2 - 400)
  root.addChild(recHeader)
  const recTitle = createLabelNode('RecTitle', '🔥 热门推荐', 17, LobbyTheme.text, 160, 28)
  recTitle.getComponent(Label)!.isBold = true
  place(recTitle, -W / 2 + PAD + 80, 0)
  recHeader.addChild(recTitle)
  const moreRec = createLabelNode('MoreRec', '更多', 12, LobbyTheme.gold, 48, 28)
  moreRec.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.RIGHT
  place(moreRec, W / 2 - PAD - 24, 0)
  recHeader.addChild(moreRec)
  moreRec.on(Node.EventType.TOUCH_END, () => handlers.onEditGames())

  const recommendRoot = createUINode('RecommendRoot')
  recommendRoot.addComponent(UITransform).setContentSize(W - PAD * 2, 90)
  place(recommendRoot, 0, H / 2 - 470)
  root.addChild(recommendRoot)

  // ===== 最近玩过 =====
  const recentHeader = createUINode('RecentHeader')
  recentHeader.addComponent(UITransform).setContentSize(W - PAD * 2, 28)
  place(recentHeader, 0, H / 2 - 540)
  root.addChild(recentHeader)
  const recentTitle = createLabelNode('RecentTitle', '⏱️ 最近玩过', 17, LobbyTheme.text, 160, 28)
  recentTitle.getComponent(Label)!.isBold = true
  place(recentTitle, -W / 2 + PAD + 80, 0)
  recentHeader.addChild(recentTitle)

  const recentRoot = createUINode('RecentRoot')
  recentRoot.addComponent(UITransform).setContentSize(W - PAD * 2, 44)
  place(recentRoot, 0, H / 2 - 590)
  root.addChild(recentRoot)

  const shelfHost = createUINode('GameShelfHost')
  shelfHost.addComponent(UITransform).setContentSize(4, 4)
  place(shelfHost, 0, 0)
  root.addChild(shelfHost)
  const gameShelf = shelfHost.addComponent(GameShelf)
  gameShelf.gridRoot = gridRoot
  gameShelf.recommendRoot = recommendRoot
  gameShelf.recentRoot = recentRoot
  gameShelf.titleLabel = shelfTitle.getComponent(Label)
  gameShelf.columns = 4
  gameShelf.cardWidth = 150
  gameShelf.cardHeight = 100
  gameShelf.cardGapX = 12
  gameShelf.cardGapY = 12

  // ===== 俱乐部底栏 =====
  const clubFooter = createUINode('ClubFooter')
  clubFooter.addComponent(UITransform).setContentSize(W - PAD * 2, 56)
  paintRoundRect(clubFooter, W - PAD * 2, 56, LobbyTheme.cardBg, 16)
  place(clubFooter, 0, -H / 2 + 56)
  root.addChild(clubFooter)
  const clubIcon = createLabelNode('ClubIcon', '🏠', 20, LobbyTheme.text, 32, 32)
  place(clubIcon, -W / 2 + PAD + 36, 0)
  clubFooter.addChild(clubIcon)
  const clubLabelNode = createLabelNode('ClubLabel', '星辰棋牌社', 14, LobbyTheme.text, 160, 28)
  place(clubLabelNode, -W / 2 + PAD + 140, 0)
  clubFooter.addChild(clubLabelNode)
  const roleBg = createUINode('RoleBg')
  roleBg.addComponent(UITransform).setContentSize(64, 22)
  paintRoundRect(roleBg, 64, 22, LobbyTheme.roomCardBg, 11)
  place(roleBg, 40, 0)
  clubFooter.addChild(roleBg)
  const clubRole = createLabelNode('ClubRole', '管理员', 11, LobbyTheme.gold, 64, 22)
  clubRole.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(clubRole, 0, 0)
  roleBg.addChild(clubRole)

  const clubArrow = createLabelNode('Arrow', '›', 18, LobbyTheme.iconMuted, 28, 28)
  place(clubArrow, W / 2 - PAD - 36, 0)
  clubFooter.addChild(clubArrow)
  clubFooter.on(Node.EventType.TOUCH_END, handlers.onClub)

  const statusNode = createLabelNode('Status', '', 12, LobbyTheme.textMuted, W - 48, 22)
  statusNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(statusNode, 0, -H / 2 + 100)
  root.addChild(statusNode)

  // ===== Toast =====
  const toastRoot = createUINode('Toast')
  toastRoot.addComponent(UITransform).setContentSize(300, 80)
  paintRoundRect(toastRoot, 300, 80, LobbyTheme.toastBg, 16)
  place(toastRoot, 0, 0)
  toastRoot.active = false
  root.addChild(toastRoot)
  const toastLabelNode = createLabelNode('ToastText', '', 15, LobbyTheme.toastText, 260, 28)
  toastLabelNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(toastLabelNode, 0, 12)
  toastRoot.addChild(toastLabelNode)
  const toastSubNode = createLabelNode('ToastSub', '', 12, LobbyTheme.textMuted, 260, 22)
  toastSubNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(toastSubNode, 0, -14)
  toastRoot.addChild(toastSubNode)

  // ===== 伴侣抽屉 =====
  const companionRoot = createUINode('CompanionPanelRoot')
  companionRoot.addComponent(UITransform).setContentSize(W - PAD * 2, 560)
  paintRoundRect(companionRoot, W - PAD * 2, 560, LobbyTheme.surface, 20)
  place(companionRoot, 0, -40)
  companionRoot.active = false
  root.addChild(companionRoot)
  const companionTitle = createLabelNode('CTitle', '❤️ 小龟伴侣', 18, LobbyTheme.brand, 280, 30)
  companionTitle.getComponent(Label)!.isBold = true
  place(companionTitle, 0, 240)
  companionRoot.addChild(companionTitle)
  const chatNode = createLabelNode('Chat', '', 14, LobbyTheme.text, W - 64, 320)
  const chatLabel = chatNode.getComponent(Label)!
  chatLabel.overflow = Label.Overflow.SHRINK
  chatLabel.verticalAlign = Label.VerticalAlign.TOP
  chatLabel.enableWrapText = true
  place(chatNode, 0, 20)
  companionRoot.addChild(chatNode)
  const quickRoot = createUINode('QuickActions')
  quickRoot.addComponent(UITransform).setContentSize(W - 64, 80)
  place(quickRoot, 0, -220)
  companionRoot.addChild(quickRoot)
  const companionPanel = companionRoot.addComponent(CompanionPanel)
  companionPanel.chatLabel = chatLabel
  companionPanel.quickActionsRoot = quickRoot

  // ===== 六子冲匹配面板 =====
  const matchRoot = createUINode('MatchPanel')
  matchRoot.addComponent(UITransform).setContentSize(W - 48, 360)
  paintRoundRect(matchRoot, W - 48, 360, LobbyTheme.surface, 20)
  place(matchRoot, 0, 20)
  matchRoot.active = false
  root.addChild(matchRoot)
  const matchTitle = createLabelNode('MatchTitle', '六子冲', 20, LobbyTheme.brand, 280, 32)
  matchTitle.getComponent(Label)!.isBold = true
  matchTitle.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(matchTitle, 0, 140)
  matchRoot.addChild(matchTitle)
  const matchHint = createLabelNode(
    'MatchHint',
    '人机开打 / 创建房间等好友 / 输入房号加入',
    13,
    LobbyTheme.textMuted,
    W - 80,
    28,
  )
  matchHint.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(matchHint, 0, 100)
  matchRoot.addChild(matchHint)

  const vsBotBtn = createMatchBtn('VsBot', '人机对战', LobbyTheme.gold)
  place(vsBotBtn, 0, 50)
  matchRoot.addChild(vsBotBtn)
  vsBotBtn.on(Node.EventType.TOUCH_END, handlers.onMatchVsBot)

  const createPvpBtn = createMatchBtn('CreatePvp', '创建房间（等人）', LobbyTheme.brand)
  place(createPvpBtn, 0, -10)
  matchRoot.addChild(createPvpBtn)
  createPvpBtn.on(Node.EventType.TOUCH_END, handlers.onMatchCreatePvp)

  const joinField = createUINode('JoinField')
  joinField.addComponent(UITransform).setContentSize(W - 100, 44)
  paintRoundRect(joinField, W - 100, 44, LobbyTheme.inputBg, 12)
  place(joinField, 0, -70)
  matchRoot.addChild(joinField)
  const joinEditNode = createUINode('JoinEdit')
  joinEditNode.addComponent(UITransform).setContentSize(W - 120, 40)
  place(joinEditNode, 0, 0)
  joinField.addChild(joinEditNode)
  const joinEdit = joinEditNode.addComponent(EditBox)
  joinEdit.placeholder = '粘贴房间 ID'
  joinEdit.maxLength = 64
  joinEdit.inputMode = EditBox.InputMode.SINGLE_LINE
  joinEdit.string = ''
  const joinText = createUINode('TEXT_LABEL')
  joinText.addComponent(UITransform).setContentSize(W - 120, 40)
  const joinTextLabel = joinText.addComponent(Label)
  joinTextLabel.fontSize = 14
  joinTextLabel.color = LobbyTheme.text
  joinTextLabel.horizontalAlign = Label.HorizontalAlign.CENTER
  joinTextLabel.verticalAlign = Label.VerticalAlign.CENTER
  joinEditNode.addChild(joinText)
  joinEdit.textLabel = joinTextLabel
  const joinPh = createUINode('PLACEHOLDER_LABEL')
  joinPh.addComponent(UITransform).setContentSize(W - 120, 40)
  const joinPhLabel = joinPh.addComponent(Label)
  joinPhLabel.string = '粘贴房间 ID'
  joinPhLabel.fontSize = 14
  joinPhLabel.color = LobbyTheme.textFaint
  joinPhLabel.horizontalAlign = Label.HorizontalAlign.CENTER
  joinPhLabel.verticalAlign = Label.VerticalAlign.CENTER
  joinEditNode.addChild(joinPh)
  joinEdit.placeholderLabel = joinPhLabel

  const joinBtn = createMatchBtn('JoinBtn', '加入房间', LobbyTheme.goldDim)
  place(joinBtn, 0, -130)
  matchRoot.addChild(joinBtn)
  joinBtn.on(Node.EventType.TOUCH_END, handlers.onMatchJoin)

  const closeMatch = createLabelNode('CloseMatch', '关闭', 14, LobbyTheme.textMuted, 80, 28)
  closeMatch.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(closeMatch, 0, -165)
  matchRoot.addChild(closeMatch)
  closeMatch.on(Node.EventType.TOUCH_END, handlers.onMatchClose)

  ensureUILayer(root)

  return {
    root,
    userLabel: userNameNode.getComponent(Label)!,
    balanceLabel: balNode.getComponent(Label)!,
    onlineLabel: onlineNode.getComponent(Label)!,
    statusLabel: statusNode.getComponent(Label)!,
    toastRoot,
    toastLabel: toastLabelNode.getComponent(Label)!,
    toastSubLabel: toastSubNode.getComponent(Label)!,
    avatarMenuRoot: avatarMenu,
    companionPanelRoot: companionRoot,
    companionBadgeLabel: badgeNode.getComponent(Label)!,
    clubLabel: clubLabelNode.getComponent(Label)!,
    gameShelf,
    companionPanel,
    matchPanelRoot: matchRoot,
    matchRoomInput: joinEdit,
    matchTitleLabel: matchTitle.getComponent(Label)!,
  }
}

function createMatchBtn(name: string, text: string, bg: Color): Node {
  const node = createUINode(name)
  node.addComponent(UITransform).setContentSize(280, 44)
  paintRoundRect(node, 280, 44, bg, 22)
  const label = createLabelNode('L', text, 16, Color.WHITE, 260, 36)
  label.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  label.getComponent(Label)!.isBold = true
  place(label, 0, 0)
  node.addChild(label)
  return node
}

function createMenuItem(name: string, text: string, color: Color, onClick: () => void): Node {
  const node = createLabelNode(name, text, 14, color, 140, 36)
  node.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  node.on(Node.EventType.TOUCH_END, (e: EventTouch) => {
    e.propagationStopped = true
    onClick()
  })
  return node
}
