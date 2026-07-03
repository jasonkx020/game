import { BinaryWriter } from '@bufbuild/protobuf/wire'

export function encodeProto<T>(
  encoder: { encode: (msg: T, writer?: BinaryWriter) => BinaryWriter },
  message: T,
): Uint8Array {
  return encoder.encode(message).finish()
}

export function decodeProto<T>(
  decoder: { decode: (input: Uint8Array) => T },
  data: Uint8Array,
): T {
  return decoder.decode(data)
}
