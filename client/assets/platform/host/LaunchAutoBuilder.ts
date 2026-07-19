import { EditBox, Label, Node, UITransform } from 'cc'
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
} from '../lobby/LobbyUIKit'

export interface LaunchBuiltUI {
  root: Node
  phoneInput: EditBox
  codeInput: EditBox
  loginBtnLabel: Label
  loginBtnNode: Node
  toastRoot: Node
  toastLabel: Label
  toastSubLabel: Label
  rememberLabel: Label
}

type Handlers = {
  onLogin: () => void
  onForgot: () => void
  onRegister: () => void
  onSocial: (name: string) => void
  onToggleRemember: () => void
}

const W = UI_DESIGN_WIDTH
const H = UI_DESIGN_HEIGHT
/** 对齐 HTML padding：左右 28、内容区宽度 */
const PAD_X = 28
const CONTENT_W = W - PAD_X * 2
const FIELD_H = 52

/**
 * 严格按登录 HTML 分区：Header → Form → Options → LoginBtn → Social → Register → Toast。
 * 第二字段对接短信验证码 API（HTML 为密码框）。
 */
export function buildLaunchUI(parent: Node, handlers: Handlers): LaunchBuiltUI {
  const old = parent.getChildByName('AutoLaunch')
  if (old) old.destroy()

  const root = createUINode('AutoLaunch')
  root.addComponent(UITransform).setContentSize(W, H)
  parent.addChild(root)

  paintSkyBackground(root, W, H)

  // ===== 1. Header =====
  const header = createUINode('LoginHeader')
  header.addComponent(UITransform).setContentSize(CONTENT_W, 80)
  place(header, 0, H / 2 - 100)
  root.addChild(header)

  const logoIcon = createLabelNode('LogoIcon', '🎮', 36, LobbyTheme.brand, 48, 48)
  logoIcon.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(logoIcon, -CONTENT_W / 2 + 28, 16)
  header.addChild(logoIcon)

  const logoText = createLabelNode('LogoText', '欢乐大厅', 26, LobbyTheme.brand, 220, 36)
  logoText.getComponent(Label)!.isBold = true
  place(logoText, -CONTENT_W / 2 + 160, 16)
  header.addChild(logoText)

  const subtitle = createLabelNode(
    'Subtitle',
    '登录即享海量棋牌 · 好友对战',
    14,
    LobbyTheme.textMuted,
    CONTENT_W - 16,
    24,
  )
  place(subtitle, 8, -22)
  header.addChild(subtitle)

  // ===== 2. Form =====
  const form = createUINode('LoginForm')
  form.addComponent(UITransform).setContentSize(CONTENT_W, 360)
  place(form, 0, H / 2 - 360)
  root.addChild(form)

  const phoneLabel = createLabelNode('PhoneLabel', '手机号 / 账号', 13, LobbyTheme.textLabel, 160, 22)
  phoneLabel.getComponent(Label)!.isBold = true
  place(phoneLabel, -CONTENT_W / 2 + 88, 150)
  form.addChild(phoneLabel)
  const { wrap: phoneWrap, edit: phoneInput } = createEditField(
    'PhoneField',
    '📱',
    '请输入手机号或账号',
    '13800000001',
  )
  place(phoneWrap, 0, 108)
  form.addChild(phoneWrap)

  const codeLabel = createLabelNode('CodeLabel', '验证码', 13, LobbyTheme.textLabel, 120, 22)
  codeLabel.getComponent(Label)!.isBold = true
  place(codeLabel, -CONTENT_W / 2 + 70, 40)
  form.addChild(codeLabel)
  const { wrap: codeWrap, edit: codeInput } = createEditField(
    'CodeField',
    '🔒',
    '开发环境默认 123456',
    '123456',
  )
  place(codeWrap, 0, -2)
  form.addChild(codeWrap)

  // ===== 3. Options =====
  const options = createUINode('FormOptions')
  options.addComponent(UITransform).setContentSize(CONTENT_W, 28)
  place(options, 0, -70)
  form.addChild(options)

  const remember = createLabelNode('Remember', '☑ 记住我', 13, LobbyTheme.textMuted, 120, 28)
  place(remember, -CONTENT_W / 2 + 70, 0)
  options.addChild(remember)
  remember.on(Node.EventType.TOUCH_END, handlers.onToggleRemember)

  const forgot = createLabelNode('Forgot', '忘记密码？', 13, LobbyTheme.gold, 100, 28)
  forgot.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.RIGHT
  place(forgot, CONTENT_W / 2 - 60, 0)
  options.addChild(forgot)
  forgot.on(Node.EventType.TOUCH_END, handlers.onForgot)

  // ===== 4. Login button =====
  const loginBtn = createUINode('LoginBtn')
  loginBtn.addComponent(UITransform).setContentSize(CONTENT_W, 54)
  // HTML: linear-gradient(#60a5fa → #3b82f6)，扁平取中间偏终点蓝
  paintRoundRect(loginBtn, CONTENT_W, 54, LobbyTheme.gold, 16)
  place(loginBtn, 0, -140)
  form.addChild(loginBtn)
  const loginBtnLabelNode = createLabelNode('LoginBtnText', '登 录', 18, LobbyTheme.btnText, 200, 36)
  const loginBtnLabel = loginBtnLabelNode.getComponent(Label)!
  loginBtnLabel.horizontalAlign = Label.HorizontalAlign.CENTER
  loginBtnLabel.isBold = true
  place(loginBtnLabelNode, 0, 0)
  loginBtn.addChild(loginBtnLabelNode)
  loginBtn.on(Node.EventType.TOUCH_END, handlers.onLogin)

  // ===== 5. Social =====
  const social = createUINode('SocialLogin')
  social.addComponent(UITransform).setContentSize(CONTENT_W, 120)
  place(social, 0, -H / 2 + 220)
  root.addChild(social)

  const divider = createLabelNode('Divider', '— 其他方式 —', 12, LobbyTheme.divider, CONTENT_W, 24)
  divider.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(divider, 0, 40)
  social.addChild(divider)

  const socials: Array<[string, string]> = [
    ['WeChat', '💬'],
    ['QQ', '🐧'],
    ['Apple', '🍎'],
  ]
  socials.forEach(([name, icon], i) => {
    const btn = createUINode(name)
    btn.addComponent(UITransform).setContentSize(48, 48)
    paintRoundRect(btn, 48, 48, LobbyTheme.panel, 24)
    place(btn, (i - 1) * 72, -16)
    social.addChild(btn)
    const ic = createLabelNode('i', icon, 26, LobbyTheme.textLabel, 40, 40)
    ic.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
    btn.addChild(ic)
    btn.on(Node.EventType.TOUCH_END, () => handlers.onSocial(name))
  })

  // ===== 6. Register tip =====
  const regTip = createLabelNode('RegTip', '还没有账号？ 立即注册', 14, LobbyTheme.textMuted, CONTENT_W, 28)
  regTip.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(regTip, 0, -H / 2 + 100)
  root.addChild(regTip)
  const regLink = createLabelNode('RegLink', '立即注册', 14, LobbyTheme.gold, 100, 28)
  regLink.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  regLink.getComponent(Label)!.isBold = true
  place(regLink, 90, 0)
  // 整行可点
  regTip.on(Node.EventType.TOUCH_END, handlers.onRegister)
  regLink.on(Node.EventType.TOUCH_END, handlers.onRegister)
  regTip.addChild(regLink)

  // ===== 7. Toast（白底深字）=====
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
  const toastSubNode = createLabelNode('ToastSub', '', 12, LobbyTheme.iconMuted, 260, 22)
  toastSubNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(toastSubNode, 0, -14)
  toastRoot.addChild(toastSubNode)

  ensureUILayer(root)

  return {
    root,
    phoneInput,
    codeInput,
    loginBtnLabel,
    loginBtnNode: loginBtn,
    toastRoot,
    toastLabel: toastLabelNode.getComponent(Label)!,
    toastSubLabel: toastSubNode.getComponent(Label)!,
    rememberLabel: remember.getComponent(Label)!,
  }
}

function createEditField(
  name: string,
  icon: string,
  placeholder: string,
  defaultValue: string,
): { wrap: Node; edit: EditBox } {
  const wrap = createUINode(name)
  const fieldW = CONTENT_W
  wrap.addComponent(UITransform).setContentSize(fieldW, FIELD_H)
  paintRoundRect(wrap, fieldW, FIELD_H, LobbyTheme.inputBg, 16)

  const iconNode = createLabelNode('Icon', icon, 18, LobbyTheme.iconMuted, 32, 32)
  iconNode.getComponent(Label)!.horizontalAlign = Label.HorizontalAlign.CENTER
  place(iconNode, -fieldW / 2 + 28, 0)
  wrap.addChild(iconNode)

  const editNode = createUINode('Edit')
  editNode.addComponent(UITransform).setContentSize(fieldW - 70, FIELD_H - 8)
  place(editNode, 16, 0)
  wrap.addChild(editNode)

  const edit = editNode.addComponent(EditBox)
  edit.string = defaultValue
  edit.placeholder = placeholder
  edit.maxLength = 20
  edit.inputMode = EditBox.InputMode.SINGLE_LINE
  edit.inputFlag = EditBox.InputFlag.DEFAULT
  edit.returnType = EditBox.KeyboardReturnType.DONE

  const textLabelNode = createUINode('TEXT_LABEL')
  textLabelNode.addComponent(UITransform).setContentSize(fieldW - 70, FIELD_H - 8)
  const textLabel = textLabelNode.addComponent(Label)
  textLabel.string = defaultValue
  textLabel.fontSize = 16
  textLabel.color = LobbyTheme.text
  textLabel.horizontalAlign = Label.HorizontalAlign.LEFT
  textLabel.verticalAlign = Label.VerticalAlign.CENTER
  editNode.addChild(textLabelNode)
  edit.textLabel = textLabel

  const phNode = createUINode('PLACEHOLDER_LABEL')
  phNode.addComponent(UITransform).setContentSize(fieldW - 70, FIELD_H - 8)
  const phLabel = phNode.addComponent(Label)
  phLabel.string = placeholder
  phLabel.fontSize = 15
  phLabel.color = LobbyTheme.textFaint
  phLabel.horizontalAlign = Label.HorizontalAlign.LEFT
  phLabel.verticalAlign = Label.VerticalAlign.CENTER
  editNode.addChild(phNode)
  edit.placeholderLabel = phLabel

  return { wrap, edit }
}
