/**
 * @deprecated 使用 LobbyScene。保留兼容旧场景绑定。
 */
import { _decorator } from 'cc'
import { LobbyScene } from './LobbyScene'

const { ccclass } = _decorator

@ccclass('HallScene')
export class HallScene extends LobbyScene {}
