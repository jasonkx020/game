/** Pitaya Pomelo packet + message codec (aligned with pitaya/v2 conn/codec + conn/message). */

export enum PacketType {
  Handshake = 0x01,
  HandshakeAck = 0x02,
  Heartbeat = 0x03,
  Data = 0x04,
  Kick = 0x05,
}

export enum MessageType {
  Request = 0x00,
  Notify = 0x01,
  Response = 0x02,
  Push = 0x03,
}

export interface PitayaMessage {
  type: MessageType
  id: number
  route: string
  data: Uint8Array
  err?: boolean
}

const HEAD_LENGTH = 4
const MSG_TYPE_MASK = 0x07
const ERROR_MASK = 0x20
const GZIP_MASK = 0x10
const ROUTE_COMPRESS_MASK = 0x01

export function encodeVarint(n: number): Uint8Array {
  const out: number[] = []
  let v = n
  while (true) {
    const b = v % 128
    v = Math.floor(v / 128)
    if (v !== 0) {
      out.push(b + 128)
    } else {
      out.push(b)
      break
    }
  }
  return Uint8Array.from(out)
}

export function decodeVarint(data: Uint8Array, offset: number): { value: number; offset: number } {
  let id = 0
  for (let i = offset; i < data.length; i++) {
    const b = data[i]
    id += (b & 0x7f) << (7 * (i - offset))
    if (b < 128) {
      return { value: id, offset: i + 1 }
    }
  }
  throw new Error('invalid varint')
}

export function encodeMessage(msg: PitayaMessage): Uint8Array {
  let flag = (msg.type << 1) & 0xff
  if (msg.err) flag |= ERROR_MASK

  const parts: number[] = [flag]

  if (msg.type === MessageType.Request || msg.type === MessageType.Response) {
    parts.push(...encodeVarint(msg.id))
  }

  if (msg.type === MessageType.Request || msg.type === MessageType.Notify || msg.type === MessageType.Push) {
    const routeBytes = new TextEncoder().encode(msg.route)
    parts.push(routeBytes.length)
    parts.push(...routeBytes)
  }

  parts.push(...msg.data)
  return Uint8Array.from(parts)
}

export function decodeMessage(data: Uint8Array): PitayaMessage {
  if (data.length < 1) throw new Error('invalid message')
  const flag = data[0]
  let offset = 1
  const type = (flag >> 1) & MSG_TYPE_MASK

  const msg: PitayaMessage = {
    type,
    id: 0,
    route: '',
    data: new Uint8Array(),
    err: (flag & ERROR_MASK) === ERROR_MASK,
  }

  if (type === MessageType.Request || type === MessageType.Response) {
    const parsed = decodeVarint(data, offset)
    msg.id = parsed.value
    offset = parsed.offset
  }

  if (type === MessageType.Request || type === MessageType.Notify || type === MessageType.Push) {
    if ((flag & ROUTE_COMPRESS_MASK) === ROUTE_COMPRESS_MASK) {
      throw new Error('compressed route not supported')
    }
    const rl = data[offset]
    offset += 1
    msg.route = new TextDecoder().decode(data.subarray(offset, offset + rl))
    offset += rl
  }

  msg.data = data.subarray(offset)
  if ((flag & GZIP_MASK) === GZIP_MASK) {
    throw new Error('gzip message not supported')
  }
  return msg
}

export function encodePacket(type: PacketType, payload: Uint8Array): Uint8Array {
  const buf = new Uint8Array(HEAD_LENGTH + payload.length)
  buf[0] = type
  const len = payload.length
  buf[1] = (len >> 16) & 0xff
  buf[2] = (len >> 8) & 0xff
  buf[3] = len & 0xff
  buf.set(payload, HEAD_LENGTH)
  return buf
}

export interface DecodedPacket {
  type: PacketType
  data: Uint8Array
}

export function decodePackets(buffer: Uint8Array): { packets: DecodedPacket[]; rest: Uint8Array } {
  const packets: DecodedPacket[] = []
  let offset = 0
  while (offset + HEAD_LENGTH <= buffer.length) {
    const type = buffer[offset] as PacketType
    const size = (buffer[offset + 1] << 16) | (buffer[offset + 2] << 8) | buffer[offset + 3]
    if (offset + HEAD_LENGTH + size > buffer.length) break
    const data = buffer.subarray(offset + HEAD_LENGTH, offset + HEAD_LENGTH + size)
    packets.push({ type, data })
    offset += HEAD_LENGTH + size
  }
  return { packets, rest: buffer.subarray(offset) }
}

export function buildHandshakePacket(): Uint8Array {
  const body = new TextEncoder().encode(JSON.stringify({ sys: { platform: 'cocos', version: '0.1.0' } }))
  return encodePacket(PacketType.Handshake, body)
}

export function buildHandshakeAckPacket(): Uint8Array {
  return encodePacket(PacketType.HandshakeAck, new Uint8Array())
}

export function buildHeartbeatPacket(): Uint8Array {
  return encodePacket(PacketType.Heartbeat, new Uint8Array())
}

export function buildDataRequest(id: number, route: string, data: Uint8Array): Uint8Array {
  const msg = encodeMessage({ type: MessageType.Request, id, route, data })
  return encodePacket(PacketType.Data, msg)
}
