import { _decorator, Component, EditBox, Label, Node, UITransform, Vec3 } from 'cc'
import { CompanionClient, type CompanionMessage } from './CompanionClient'
import { SessionStore } from '../../scripts/SessionStore'

const { ccclass, property } = _decorator

@ccclass('CompanionPanel')
export class CompanionPanel extends Component {
  @property(EditBox)
  inputBox: EditBox | null = null

  @property(Label)
  chatLabel: Label | null = null

  @property(Node)
  quickActionsRoot: Node | null = null

  private client: CompanionClient | null = null
  private sessionId = 0
  private lines: string[] = []
  private sending = false

  async init(getToken: () => string): Promise<void> {
    this.client = new CompanionClient(getToken)
    const sess = await this.client.createSession('default')
    this.sessionId = sess.id
    SessionStore.companionSessionId = sess.id
    this.appendLine('小龟', '嗨！我是你的陪玩伴侣～想聊天、学规则还是开一局？')
    this.renderQuickActions()
  }

  getSessionId(): number {
    return this.sessionId
  }

  async onSendClick(): Promise<void> {
    const text = this.inputBox?.string?.trim()
    if (!text || !this.client || this.sending) return
    this.sending = true
    this.appendLine('我', text)
    if (this.inputBox) this.inputBox.string = ''
    let assistant = ''
    try {
      await this.client.chatStream(this.sessionId, text, (chunk, done) => {
        if (chunk) assistant += chunk
        if (done) {
          this.appendLine('小龟', assistant || '…')
        }
        this.refreshChat()
      })
    } catch (e) {
      this.appendLine('小龟', '网络有点卡，稍后再聊～')
      console.error('[Companion]', e)
    } finally {
      this.sending = false
    }
  }

  async sendQuick(text: string): Promise<void> {
    if (this.inputBox) this.inputBox.string = text
    await this.onSendClick()
  }

  private renderQuickActions(): void {
    if (!this.quickActionsRoot) return
    this.quickActionsRoot.removeAllChildren()
    const actions = ['推荐游戏', '讲讲打乌龟规则', '开一局4人打乌龟']
    actions.forEach((text, index) => {
      const row = new Node(`quick-${index}`)
      row.addComponent(UITransform).setContentSize(200, 36)
      row.setPosition(new Vec3((index % 2) * 220 - 110, -Math.floor(index / 2) * 40, 0))
      const label = row.addComponent(Label)
      label.string = `[${text}]`
      label.fontSize = 18
      row.on(Node.EventType.TOUCH_END, () => void this.sendQuick(text))
      this.quickActionsRoot.addChild(row)
    })
  }

  private appendLine(who: string, text: string): void {
    this.lines.push(`${who}: ${text}`)
    if (this.lines.length > 12) this.lines.shift()
    this.refreshChat()
  }

  private refreshChat(): void {
    if (this.chatLabel) this.chatLabel.string = this.lines.join('\n')
  }
}

export type { CompanionMessage }
