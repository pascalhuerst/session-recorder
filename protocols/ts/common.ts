/* eslint-disable */
import _m0 from "protobufjs/minimal";

export const protobufPackage = "common";

export enum AudioInputStatus {
  UNKNOWN = 0,
  NO_SIGNAL = 1,
  SIGNAL = 2,
  UNRECOGNIZED = -1,
}

export function audioInputStatusFromJSON(object: any): AudioInputStatus {
  switch (object) {
    case 0:
    case "UNKNOWN":
      return AudioInputStatus.UNKNOWN;
    case 1:
    case "NO_SIGNAL":
      return AudioInputStatus.NO_SIGNAL;
    case 2:
    case "SIGNAL":
      return AudioInputStatus.SIGNAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return AudioInputStatus.UNRECOGNIZED;
  }
}

export function audioInputStatusToJSON(object: AudioInputStatus): string {
  switch (object) {
    case AudioInputStatus.UNKNOWN:
      return "UNKNOWN";
    case AudioInputStatus.NO_SIGNAL:
      return "NO_SIGNAL";
    case AudioInputStatus.SIGNAL:
      return "SIGNAL";
    case AudioInputStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}

export interface Respone {
  success: boolean;
  errorMessage: string;
}

function createBaseRespone(): Respone {
  return { success: false, errorMessage: "" };
}

export const Respone = {
  encode(message: Respone, writer: _m0.Writer = _m0.Writer.create()): _m0.Writer {
    if (message.success === true) {
      writer.uint32(8).bool(message.success);
    }
    if (message.errorMessage !== "") {
      writer.uint32(18).string(message.errorMessage);
    }
    return writer;
  },

  decode(input: _m0.Reader | Uint8Array, length?: number): Respone {
    const reader = input instanceof _m0.Reader ? input : _m0.Reader.create(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseRespone();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          if (tag !== 8) {
            break;
          }

          message.success = reader.bool();
          continue;
        case 2:
          if (tag !== 18) {
            break;
          }

          message.errorMessage = reader.string();
          continue;
      }
      if ((tag & 7) === 4 || tag === 0) {
        break;
      }
      reader.skipType(tag & 7);
    }
    return message;
  },

  fromJSON(object: any): Respone {
    return {
      success: isSet(object.success) ? globalThis.Boolean(object.success) : false,
      errorMessage: isSet(object.errorMessage) ? globalThis.String(object.errorMessage) : "",
    };
  },

  toJSON(message: Respone): unknown {
    const obj: any = {};
    if (message.success === true) {
      obj.success = message.success;
    }
    if (message.errorMessage !== "") {
      obj.errorMessage = message.errorMessage;
    }
    return obj;
  },

  create(base?: DeepPartial<Respone>): Respone {
    return Respone.fromPartial(base ?? {});
  },
  fromPartial(object: DeepPartial<Respone>): Respone {
    const message = createBaseRespone();
    message.success = object.success ?? false;
    message.errorMessage = object.errorMessage ?? "";
    return message;
  },
};

type Builtin = Date | Function | Uint8Array | string | number | boolean | undefined;

export type DeepPartial<T> = T extends Builtin ? T
  : T extends globalThis.Array<infer U> ? globalThis.Array<DeepPartial<U>>
  : T extends ReadonlyArray<infer U> ? ReadonlyArray<DeepPartial<U>>
  : T extends {} ? { [K in keyof T]?: DeepPartial<T[K]> }
  : Partial<T>;

function isSet(value: any): boolean {
  return value !== null && value !== undefined;
}
