import { _decorator, Component, director, EditBox } from 'cc'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

@ccclass('LaunchScene')
export class LaunchScene extends Component {
  @property(EditBox)
  phoneInput: EditBox | null = null

  @property(EditBox)
  codeInput: EditBox | null = null

  async onLoginClick(): Promise<void> {
    const phone = this.phoneInput?.string || '13800000001'
    const code = this.codeInput?.string || '123456'
    try {
      SessionStore.login = await SessionStore.api.login(phone, code)
      director.loadScene('Hall')
    } catch (e) {
      console.error('[Launch] login failed', e)
    }
  }
}
