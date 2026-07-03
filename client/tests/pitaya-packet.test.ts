import { describe, expect, it } from 'vitest'
import {
  MessageType,
  decodeMessage,
  decodePackets,
  encodeMessage,
  encodePacket,
  buildDataRequest,
  PacketType,
} from '../assets/platform/sdk/PitayaPacket'

describe('PitayaPacket', () => {
  it('round-trips request message', () => {
    const route = 'game.connector.entry'
    const data = new Uint8Array([1, 2, 3])
    const encoded = encodeMessage({ type: MessageType.Request, id: 1, route, data })
    const decoded = decodeMessage(encoded)
    expect(decoded.type).toBe(MessageType.Request)
    expect(decoded.id).toBe(1)
    expect(decoded.route).toBe(route)
    expect([...decoded.data]).toEqual([1, 2, 3])
  })

  it('round-trips push message', () => {
    const route = 'onDeal'
    const data = new Uint8Array([9])
    const encoded = encodeMessage({ type: MessageType.Push, id: 0, route, data })
    const decoded = decodeMessage(encoded)
    expect(decoded.type).toBe(MessageType.Push)
    expect(decoded.route).toBe(route)
    expect(decoded.data[0]).toBe(9)
  })

  it('encodes data packet with header', () => {
    const packet = buildDataRequest(2, 'game.room.join', new Uint8Array([0]))
    const { packets, rest } = decodePackets(packet)
    expect(rest.length).toBe(0)
    expect(packets.length).toBe(1)
    expect(packets[0].type).toBe(PacketType.Data)
    const msg = decodeMessage(packets[0].data)
    expect(msg.route).toBe('game.room.join')
    expect(msg.id).toBe(2)
  })

  it('decodes multiple packets in buffer', () => {
    const p1 = encodePacket(PacketType.Heartbeat, new Uint8Array())
    const p2 = buildDataRequest(1, 'game.connector.bind', new Uint8Array([5]))
    const merged = new Uint8Array(p1.length + p2.length)
    merged.set(p1)
    merged.set(p2, p1.length)
    const { packets } = decodePackets(merged)
    expect(packets.length).toBe(2)
    expect(packets[0].type).toBe(PacketType.Heartbeat)
    expect(packets[1].type).toBe(PacketType.Data)
  })
})
