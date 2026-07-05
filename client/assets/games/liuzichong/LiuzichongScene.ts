import { _decorator, Component } from 'cc'
import { LiuzichongBoard } from './LiuzichongBoard'

const { ccclass } = _decorator

/** Liuzichong 场景根节点：挂载 LiuzichongBoard 子组件或同节点脚本 */
@ccclass('LiuzichongScene')
export class LiuzichongScene extends Component {
  start(): void {
    if (!this.getComponent(LiuzichongBoard)) {
      this.node.addComponent(LiuzichongBoard)
    }
  }
}
