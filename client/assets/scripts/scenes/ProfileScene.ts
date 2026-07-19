import { _decorator, Component, director } from 'cc'
import { ProfilePanel } from '../../platform/lobby/ProfilePanel'
import { resolveOrCreateUICanvas } from '../../platform/lobby/UICanvasHost'
import { SessionStore } from '../SessionStore'

const { ccclass, property } = _decorator

/**
 * 可选独立 Profile 场景。大厅默认用覆盖层 ProfilePanel，无需本场景。
 * 若 Build Settings 里有 Profile 场景，挂本组件即可自动搭 UI。
 */
@ccclass('ProfileScene')
export class ProfileScene extends Component {
  @property
  autoBuildUI = true

  private panel: ProfilePanel | null = null

  onLoad(): void {
    if (!this.autoBuildUI) return
    const host = this.resolveUIHost()
    this.hideLegacySiblings(host)
    this.panel = new ProfilePanel(host, {
      onBack: () => director.loadScene('Lobby'),
      onLogout: () => {
        SessionStore.logout()
        director.loadScene('Launch')
      },
    })
  }

  async start(): Promise<void> {
    if (!SessionStore.login) {
      director.loadScene('Launch')
      return
    }
    this.panel?.show()
  }

  private resolveUIHost() {
    return resolveOrCreateUICanvas(this.node)
  }

  private hideLegacySiblings(host: typeof this.node): void {
    for (const child of [...host.children]) {
      const n = child.name
      if (n === 'Camera' || n === 'Main Camera' || n.includes('Camera')) continue
      if (child.getComponent('cc.Camera')) continue
      if (n === 'ProfileOverlay') continue
      child.active = false
    }
  }
}
