import { _decorator, Component, Label } from 'cc'
import { SessionStore } from '../../scripts/SessionStore'

const { ccclass, property } = _decorator

/** 局内侧栏：展示伴侣陪玩提示 */
@ccclass('CompanionSidebar')
export class CompanionSidebar extends Component {
  @property(Label)
  hintLabel: Label | null = null

  @property(Label)
  titleLabel: Label | null = null

  start(): void {
    if (this.titleLabel) this.titleLabel.string = '陪玩小龟'
    this.refresh()
  }

  update(): void {
    this.refresh()
  }

  private refresh(): void {
    if (!this.hintLabel) return
    const hints = SessionStore.companionHints.slice(-3)
    this.hintLabel.string = hints.length ? hints.join('\n') : '加油，我在这陪你～'
  }
}
