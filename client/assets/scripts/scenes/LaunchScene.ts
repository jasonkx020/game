import { _decorator, Component, director, EditBox, Label, Node, sys } from 'cc'
import { buildLaunchUI } from '../../platform/host/LaunchAutoBuilder'
import { LobbyTheme, setNodeOpacity } from '../../platform/lobby/LobbyUIKit'
import { resolveOrCreateUICanvas } from '../../platform/lobby/UICanvasHost'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

const REMEMBER_KEY = 'launch_remember_phone'

@ccclass('LaunchScene')
export class LaunchScene extends Component {
  /** 默认 true：代码搭建登录页，无需编辑器拖引用 */
  @property
  autoBuildUI = true

  @property(EditBox)
  phoneInput: EditBox | null = null

  @property(EditBox)
  codeInput: EditBox | null = null

  private loginBtnLabel: Label | null = null
  private loginBtnNode: Node | null = null
  private toastRoot: Node | null = null
  private toastLabel: Label | null = null
  private toastSubLabel: Label | null = null
  private rememberLabel: Label | null = null
  private remember = true
  private loggingIn = false
  private toastTimer: ReturnType<typeof setTimeout> | null = null
  private built = false

  onLoad(): void {
    if (this.autoBuildUI) this.mountAutoUI()
  }

  start(): void {
    if (this.autoBuildUI && !this.built) this.mountAutoUI()
    this.applyRememberedPhone()
  }

  private mountAutoUI(): void {
    const host = this.resolveUIHost()
    const ui = buildLaunchUI(host, {
      onLogin: () => void this.onLoginClick(),
      onForgot: () => this.showToast('重置密码', '请联系客服或通过短信找回'),
      onRegister: () =>
        this.showToast('自动注册', '新手机号登录时会自动创建玩家账号'),
      onSocial: (name) => this.showToast(`${name} 登录`, '第三方授权登录开发中'),
      onToggleRemember: () => this.toggleRemember(),
    })
    this.phoneInput = ui.phoneInput
    this.codeInput = ui.codeInput
    this.loginBtnLabel = ui.loginBtnLabel
    this.loginBtnNode = ui.loginBtnNode
    this.toastRoot = ui.toastRoot
    this.toastLabel = ui.toastLabel
    this.toastSubLabel = ui.toastSubLabel
    this.rememberLabel = ui.rememberLabel
    this.built = true
    this.hideLegacySiblings(host, ui.root)
    this.syncRememberLabel()
  }

  private resolveUIHost(): Node {
    return resolveOrCreateUICanvas(this.node)
  }

  private hideLegacySiblings(host: Node, keep: Node): void {
    for (const child of [...host.children]) {
      if (child === keep) continue
      const n = child.name
      if (n === 'Camera' || n === 'Main Camera' || n.includes('Camera')) continue
      if (child.getComponent('cc.Camera')) continue
      child.active = false
    }
  }

  private applyRememberedPhone(): void {
    if (!this.phoneInput) return
    try {
      const saved = sys.localStorage.getItem(REMEMBER_KEY)
      if (saved) this.phoneInput.string = saved
    } catch {
      /* ignore */
    }
  }

  private toggleRemember(): void {
    this.remember = !this.remember
    this.syncRememberLabel()
    if (!this.remember) {
      try {
        sys.localStorage.removeItem(REMEMBER_KEY)
      } catch {
        /* ignore */
      }
    }
  }

  private syncRememberLabel(): void {
    if (this.rememberLabel) {
      this.rememberLabel.string = this.remember ? '☑ 记住我' : '☐ 记住我'
      this.rememberLabel.color = LobbyTheme.textMuted
    }
  }

  async onLoginClick(): Promise<void> {
    if (this.loggingIn) return
    const phone = (this.phoneInput?.string || '').trim() || '13800000001'
    const code = (this.codeInput?.string || '').trim() || '123456'

    if (!phone || !code) {
      this.showToast('请填写完整信息', '手机号和验证码都不能为空')
      return
    }

    this.loggingIn = true
    this.setLoginBusy(true)
    try {
      SessionStore.login = await SessionStore.api.login(phone, code)
      if (this.remember) {
        try {
          sys.localStorage.setItem(REMEMBER_KEY, phone)
        } catch {
          /* ignore */
        }
      }
      this.showToast('登录成功', `欢迎，${SessionStore.login.nickname}`)
      director.loadScene('Lobby')
    } catch (e) {
      console.error('[Launch] login failed', e)
      const msg = e && typeof e === 'object' && 'message' in e ? String((e as { message: string }).message) : String(e)
      this.showToast('登录失败', msg)
      this.setLoginBusy(false)
      this.loggingIn = false
    }
  }

  private setLoginBusy(busy: boolean): void {
    if (this.loginBtnLabel) this.loginBtnLabel.string = busy ? '登录中...' : '登 录'
    if (this.loginBtnNode) setNodeOpacity(this.loginBtnNode, busy ? 180 : 255)
  }

  private showToast(text: string, sub = ''): void {
    if (this.toastLabel) this.toastLabel.string = text
    if (this.toastSubLabel) this.toastSubLabel.string = sub
    if (this.toastRoot) this.toastRoot.active = true
    if (this.toastTimer) clearTimeout(this.toastTimer)
    this.toastTimer = setTimeout(() => {
      if (this.toastRoot) this.toastRoot.active = false
    }, 2000)
  }

  onDestroy(): void {
    if (this.toastTimer) clearTimeout(this.toastTimer)
  }
}
